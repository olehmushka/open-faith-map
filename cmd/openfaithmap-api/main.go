// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Command openfaithmap-api is the composition root for OpenFaithMap's backend
// (docs/architecture/overview.md). `serve` boots the witchcraft server. M10.5.5 split what was one
// flat 473-line initServer function into deps.go (shared wiring state + cross-module adapters) and
// one register_<module>.go per module — see deps.go's own doc comment for why, and why now rather
// than after M10.6's cutover.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	identitymiddleware "github.com/olehmushka/open-faith-map/internal/identity/middleware"
	"github.com/olehmushka/open-faith-map/internal/platform/config"
	"github.com/olehmushka/open-faith-map/internal/platform/seed"
	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	cmd := "serve"
	if len(args) > 0 {
		cmd = args[0]
	}

	switch cmd {
	case "serve":
		return serve()
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q (known: serve)\n", cmd)
		return 2
	}
}

// serve builds the identity authenticator UNBOUND and registers its Handle method on the server
// before Start — the late-binding pattern internal/identity/middleware.Authenticator's own doc
// comment describes. Bind (wiring the real validator/resolver) happens inside initServer, once the
// DB pool and services exist; server.WithMiddleware(authenticator.Handle) is a method value closing
// over this same pointer, so the request path calls into whatever authenticator.bound holds by the
// time the first request arrives, not a snapshot taken here.
//
// M10.6: this attachment is now load-bearing for real traffic — all six consumer modules read the
// subject this middleware puts in context, and isBypassPath's extended (method, path) allowlist
// (internal/identity/middleware/authenticator.go) is what keeps every still-anonymous product
// endpoint reachable anyway.
func serve() int {
	authenticator := identitymiddleware.NewUnbound()

	server := witchcraft.NewServer().
		WithInstallConfigType(config.Install{}).
		WithRuntimeConfigType(config.Runtime{}).
		WithSelfSignedCertificate().
		WithMiddleware(authenticator.Handle).
		WithInitFunc(func(ctx context.Context, info witchcraft.InitInfo) (func(), error) {
			return initServer(ctx, info, authenticator)
		})

	if err := server.Start(); err != nil {
		// witchcraft already logged the structured error; signal non-zero exit.
		return 1
	}
	return 0
}

// registerOrder is every module's register<Module> function, in the order M10.5.5/M10.6 require:
// registerIdentity first (builds the boot-time authenticator and runs the first-admin seed, no
// routes of its own), registerCore second (M10.1-M10.5's in-process modules — directory/authz/
// religion/location/membership/refdata — every consumer module below depends on directly),
// registerReligion right after (M13.2 — its only dependency is deps.ReligionSvc, populated by
// registerCore), registerContent before registerDiscovery (discovery's constructor needs content's
// app service via deps.ContentAppSvc), registerModeration before registerVouching (same shape,
// deps.ModerationAppSvc).
var registerOrder = []struct {
	name string
	fn   registerFunc
}{
	{"identity", registerIdentity},
	{"core", registerCore},
	{"religion", registerReligion},
	{"registration", registerRegistration},
	{"content", registerContent},
	{"discovery", registerDiscovery},
	{"moderation", registerModeration},
	{"vouching", registerVouching},
	{"congregationimport", registerCongregationImport},
}

// Conservative pool-sizing defaults for a single-instance API service — not yet operator-tunable
// (no existing precedent in this codebase's config.Install/config.Runtime for exposing this class
// of knob as an env var); revisit if this ever needs to scale beyond one instance's own good sense.
const (
	poolMaxConns          = 20
	poolMinConns          = 2
	poolMaxConnLifetime   = 30 * time.Minute
	poolMaxConnIdleTime   = 5 * time.Minute
	poolHealthCheckPeriod = time.Minute
)

// initServer is the composition root's InitFunc (docs/architecture/overview.md). Dials the shared
// Postgres pool (openfaithmap schema — migrations applied out-of-band by docker-compose.yml's
// openfaithmap-migrate, never by this binary), builds Deps once, then runs every module's
// register<Module> function in registerOrder. pool.Close() is now called from exactly one place —
// here, on the error path — collapsing what was 19 repeated call sites in the pre-M10.5.5 flat
// function.
func initServer(ctx context.Context, info witchcraft.InitInfo, authenticator *identitymiddleware.Authenticator) (func(), error) {
	install, _ := info.InstallConfig.(config.Install)
	if install.DatabaseURL == "" {
		return nil, werror.ErrorWithContextParams(ctx, "install config: database-url is required")
	}

	poolCfg, err := pgxpool.ParseConfig(install.DatabaseURL)
	if err != nil {
		return nil, werror.WrapWithContextParams(ctx, err, "parse postgres config")
	}
	poolCfg.MaxConns = poolMaxConns
	poolCfg.MinConns = poolMinConns
	poolCfg.MaxConnLifetime = poolMaxConnLifetime
	poolCfg.MaxConnIdleTime = poolMaxConnIdleTime
	poolCfg.HealthCheckPeriod = poolHealthCheckPeriod
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, werror.WrapWithContextParams(ctx, err, "dial postgres")
	}

	ids, err := seed.Resolve(ctx, pool)
	if err != nil {
		pool.Close()
		return nil, werror.WrapWithContextParams(ctx, err, "resolve seed IDs")
	}

	deps := newDeps(pool, install, ids)
	deps.Authenticator = authenticator

	for _, r := range registerOrder {
		if err := r.fn(ctx, info, deps); err != nil {
			pool.Close()
			return nil, werror.WrapWithContextParams(ctx, err, "register "+r.name+" module")
		}
	}

	return pool.Close, nil
}
