// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Command conjure-ir-dump extracts the compiled Conjure IR (api/*.conjure.yml, the "platform"
// project in godel/config/conjure-plugin.yml) as raw JSON. godel compiles api/ to IR and publishes
// it; this tool captures that by standing up a tiny local sink and pointing the publish at it (no
// JVM, no network, no external tools). It feeds scripts/gen-ts-client.sh, which needs the IR to
// generate the TypeScript SDK the same way internal/conjure is generated for Go — ported from
// go-oikumenea's tools/ir2openapi (M2.6, D-Stack), trimmed to IR extraction only: this repo has no
// existing OpenAPI doc generation to preserve, so that half of the original tool is not carried over.
//
// Usage:
//
//	go run ./tools/conjure-ir-dump -out /tmp/conjure-ir.json
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	out := flag.String("out", "", "write the raw Conjure IR JSON to this path (required)")
	godelw := flag.String("godelw", "./godelw", "path to the godelw wrapper")
	flag.Parse()

	if *out == "" {
		fatal("missing required -out")
	}

	raw, err := extractIR(*godelw)
	if err != nil {
		fatal("extract IR: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fatal("mkdir %s: %v", filepath.Dir(*out), err)
	}
	if err := os.WriteFile(*out, raw, 0o644); err != nil {
		fatal("write %s: %v", *out, err)
	}
	fmt.Printf("wrote %s (%d bytes of Conjure IR)\n", *out, len(raw))
}

func fatal(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "conjure-ir-dump: "+format+"\n", a...)
	os.Exit(1)
}

// extractIR runs `godel conjure-publish` against an in-process HTTP sink and returns the captured
// Conjure IR JSON. godel compiles api/ to IR and PUTs it; we keep the platform project's IR.
func extractIR(godelw string) ([]byte, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	defer func() { _ = ln.Close() }()
	port := ln.Addr().(*net.TCPAddr).Port

	captured := map[string][]byte{}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			b, _ := io.ReadAll(r.Body)
			if strings.HasSuffix(r.URL.Path, ".conjure.json") {
				captured[r.URL.Path] = b
			}
			w.WriteHeader(http.StatusCreated)
			return
		}
		w.WriteHeader(http.StatusOK) // checksum POSTs etc.
	})
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()
	defer func() { _ = srv.Close() }()

	cmd := exec.Command(godelw, "conjure-publish",
		"--group-id", "local.ts.gen", "--no-pom",
		"--repository", "local", "--url", fmt.Sprintf("http://127.0.0.1:%d", port))
	// GIT_DIR is pointed at nothing on purpose: godel derives the product VERSION from `git describe
	// --tags`, which is meaningless here — nothing is really published, conjure-publish is only being
	// used to make godel hand over the IR, and the "repository" is the throwaway HTTP server above.
	// Denying godel a git repo to derive from makes it report `unspecified` and keeps the publish
	// path predictable, matching go-oikumenea's own tools/ir2openapi (which needed this to dodge a
	// slash-containing nested-module git tag — this repo has no such tag today, but the shim is
	// harmless either way and keeps the two tools' IR-extraction behavior identical).
	cmd.Env = append(os.Environ(), "GIT_DIR=/nonexistent-conjure-ir-dump-version-shim")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	// conjure-publish exits non-zero on the checksum step; ignore the error and rely on the capture.
	_ = cmd.Run()

	if len(captured) == 0 {
		return nil, fmt.Errorf("godel published no IR (stderr: %s)", strings.TrimSpace(stderr.String()))
	}
	// Prefer the platform project's IR; otherwise take any (they describe the same API).
	for path, body := range captured {
		if strings.Contains(path, "/platform/") {
			return body, nil
		}
	}
	for _, body := range captured {
		return body, nil
	}
	return nil, fmt.Errorf("unreachable")
}
