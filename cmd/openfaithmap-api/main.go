// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Command openfaithmap-api is the composition root for OpenFaithMap's backend
// (docs/architecture/overview.md). `serve` boots the witchcraft server. M2 added the first real
// module, registration (docs/modules/registration.md); M3 adds content (docs/modules/content.md) —
// moderation/vouching still land as each reaches its own "backend" gate, see
// docs/development-process.md.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	gencontent "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/content"
	genregistration "github.com/olehmushka/open-faith-map/internal/conjure/openfaithmap/registration"
	contentadapters "github.com/olehmushka/open-faith-map/internal/content/adapters"
	contentapplication "github.com/olehmushka/open-faith-map/internal/content/application"
	contenttransport "github.com/olehmushka/open-faith-map/internal/content/transport"
	"github.com/olehmushka/open-faith-map/internal/platform/config"
	regadapters "github.com/olehmushka/open-faith-map/internal/registration/adapters"
	regapplication "github.com/olehmushka/open-faith-map/internal/registration/application"
	regtransport "github.com/olehmushka/open-faith-map/internal/registration/transport"
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

// initServer is the composition root's InitFunc (docs/architecture/overview.md). Wires the
// registration module: a Postgres pool (openfaithmap schema — migrations applied out-of-band by
// docker-compose.yml's openfaithmap-migrate, never by this binary) and the transport/application/
// adapters chain, registered onto witchcraft's router.
func initServer(ctx context.Context, info witchcraft.InitInfo) (func(), error) {
	databaseURL := requireEnv("DATABASE_URL")
	oikumeneaBaseURL := requireEnv("OIKUMENEA_BASE_URL")
	rootUnitID := requireEnv("REGISTRATION_ROOT_UNIT_ID")
	congregationAdminRoleID := requireEnv("REGISTRATION_CONGREGATION_ADMIN_ROLE_ID")
	insecureSkipVerify := os.Getenv("OIKUMENEA_INSECURE_SKIP_VERIFY") == "true"

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, werror.WrapWithContextParams(ctx, err, "dial postgres")
	}

	store := regadapters.NewStore(pool)
	appSvc := regapplication.NewService(store, regapplication.Config{
		OikumeneaBaseURL:            oikumeneaBaseURL,
		OikumeneaInsecureSkipVerify: insecureSkipVerify,
		RootUnitID:                  rootUnitID,
		CongregationAdminRoleID:     congregationAdminRoleID,
	})
	transportSvc := regtransport.NewService(appSvc, regtransport.Config{
		OikumeneaBaseURL:            oikumeneaBaseURL,
		OikumeneaInsecureSkipVerify: insecureSkipVerify,
	})

	if err := genregistration.RegisterRoutesRegistrationService(info.Router, transportSvc); err != nil {
		pool.Close()
		return nil, werror.WrapWithContextParams(ctx, err, "register registration routes")
	}

	contentStore := contentadapters.NewStore(pool)
	contentAppSvc := contentapplication.NewService(contentStore, contentapplication.Config{
		OikumeneaBaseURL:            oikumeneaBaseURL,
		OikumeneaInsecureSkipVerify: insecureSkipVerify,
	})
	contentTransportSvc := contenttransport.NewService(contentAppSvc, contenttransport.Config{
		OikumeneaBaseURL:            oikumeneaBaseURL,
		OikumeneaInsecureSkipVerify: insecureSkipVerify,
	})
	contentPublicTransportSvc := contenttransport.NewPublicService(contentAppSvc)

	if err := gencontent.RegisterRoutesContentService(info.Router, contentTransportSvc); err != nil {
		pool.Close()
		return nil, werror.WrapWithContextParams(ctx, err, "register content routes")
	}
	if err := gencontent.RegisterRoutesContentPublicService(info.Router, contentPublicTransportSvc); err != nil {
		pool.Close()
		return nil, werror.WrapWithContextParams(ctx, err, "register content public routes")
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
