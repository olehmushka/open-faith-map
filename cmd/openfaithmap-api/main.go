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

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olehmushka/open-faith-map/internal/platform/config"
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

func serve() int {
	server := witchcraft.NewServer().
		WithInstallConfigType(config.Install{}).
		WithRuntimeConfigType(config.Runtime{}).
		WithSelfSignedCertificate().
		WithInitFunc(initServer)

	if err := server.Start(); err != nil {
		// witchcraft already logged the structured error; signal non-zero exit.
		return 1
	}
	return 0
}

// registerOrder is every module's register<Module> function, in the order M10.5.5 requires:
// registerContent before registerDiscovery (discovery's constructor needs content's app service via
// deps.ContentAppSvc), registerModeration before registerVouching (same shape, deps.ModerationAppSvc).
// registerIdentity runs first — it builds the boot-time authenticator and runs the first-admin seed,
// though it registers no HTTP routes of its own yet (identity has no Conjure surface until M10.7).
var registerOrder = []struct {
	name string
	fn   registerFunc
}{
	{"identity", registerIdentity},
	{"registration", registerRegistration},
	{"content", registerContent},
	{"discovery", registerDiscovery},
	{"moderation", registerModeration},
	{"vouching", registerVouching},
	{"congregationimport", registerCongregationImport},
}

// initServer is the composition root's InitFunc (docs/architecture/overview.md). Dials the shared
// Postgres pool (openfaithmap schema — migrations applied out-of-band by docker-compose.yml's
// openfaithmap-migrate, never by this binary), builds Deps once, then runs every module's
// register<Module> function in registerOrder. pool.Close() is now called from exactly one place —
// here, on the error path — collapsing what was 19 repeated call sites in the pre-M10.5.5 flat
// function.
func initServer(ctx context.Context, info witchcraft.InitInfo) (func(), error) {
	databaseURL := requireEnv("DATABASE_URL")
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, werror.WrapWithContextParams(ctx, err, "dial postgres")
	}

	install, _ := info.InstallConfig.(config.Install)
	deps := newDeps(pool, install)

	for _, r := range registerOrder {
		if err := r.fn(ctx, info, deps); err != nil {
			pool.Close()
			return nil, werror.WrapWithContextParams(ctx, err, "register "+r.name+" module")
		}
	}

	return pool.Close, nil
}

func requireEnv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		panic(fmt.Sprintf("missing required env var %s", name))
	}
	return v
}
