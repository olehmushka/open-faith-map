// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Command openfaithmap-api is the composition root for OpenFaithMap's backend
// (docs/architecture/overview.md). `serve` boots the witchcraft server. This is a scaffolding-stage
// skeleton: it registers no modules yet, so only witchcraft's built-in /status/liveness,
// /status/readiness, and /status/health endpoints are served. Real modules (content, discovery,
// moderation, vouching) get wired into initServer as each reaches its "backend" gate — see
// docs/development-process.md.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/olehmushka/open-faith-map/internal/platform/config"
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

// initServer is the composition root's InitFunc (docs/architecture/overview.md). Empty for now —
// no module has reached its "backend" gate yet (docs/milestones.md's stage board). Each module's
// module.go gets wired in here, in dependency order, as it lands.
func initServer(_ context.Context, _ witchcraft.InitInfo) (func(), error) {
	return nil, nil
}
