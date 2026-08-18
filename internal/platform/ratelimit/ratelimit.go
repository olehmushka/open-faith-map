// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package ratelimit is an in-process, per-(client IP, endpoint) token-bucket limiter for anonymous
// write/read endpoints (M7, docs/modules/hardening.md; moved here at M10.6 so discovery's anonymous
// Search can reuse the exact limiter moderation's two anonymous endpoints already used —
// internal/moderation/transport used to own this package-private).
package ratelimit

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/palantir/pkg/metrics"
	"github.com/palantir/witchcraft-go-server/v2/wrouter"
	"golang.org/x/time/rate"
)

// key is (client IP, endpoint) — two independent buckets per caller so one endpoint's traffic can't
// starve another's budget (D-Hardening, docs/architecture/decisions.md).
type key struct {
	ip       string
	endpoint string
}

// Limiter is an in-process, per-(client IP, endpoint) token-bucket limiter. State is single-process
// and ephemeral by design — a restart clears every bucket, and it does not coordinate across
// replicas; fine while openfaithmap-api runs single-replica (docs/modules/hardening.md's Open Seams).
//
// The limiters map never evicts entries — a long-running process accumulates one *rate.Limiter per
// distinct (IP, endpoint) pair forever. Known, accepted limitation, not solved here: there is no
// data on real request volume to size an eviction policy against yet.
type Limiter struct {
	mu       sync.Mutex
	limiters map[key]*rate.Limiter
	metric   string
}

// NewLimiter returns a Limiter that increments metric on every rejection — each caller (moderation,
// discovery) passes its own metric name so rejections stay attributable per module.
func NewLimiter(metric string) *Limiter {
	return &Limiter{limiters: make(map[key]*rate.Limiter), metric: metric}
}

func (rl *Limiter) allow(ip, endpoint string) bool {
	k := key{ip: ip, endpoint: endpoint}
	rl.mu.Lock()
	lim, ok := rl.limiters[k]
	if !ok {
		// Provisional, not data-tuned (D-Hardening): ~5 requests/minute sustained, burst of 5.
		lim = rate.NewLimiter(rate.Every(time.Minute/5), 5)
		rl.limiters[k] = lim
	}
	rl.mu.Unlock()
	return lim.Allow()
}

// Middleware implements wrouter.RouteHandlerMiddleware. Runs pre-auth, before the request reaches
// the generated Conjure handler — on reject, writes a raw 429 directly, never a Conjure-typed error
// (Conjure's fixed error-code system has no code that maps to HTTP 429; this is a deliberate,
// permanent departure from this repo's Conjure-error-body convention, not a gap to fix later).
func (rl *Limiter) Middleware(rw http.ResponseWriter, r *http.Request, reqVals wrouter.RequestVals, next wrouter.RouteRequestHandler) {
	ip := clientIP(r)
	if rl.allow(ip, r.URL.Path) {
		next(rw, r, reqVals)
		return
	}
	metrics.FromContext(r.Context()).Counter(rl.metric).Inc(1)
	rw.Header().Set("Retry-After", "12") // matches rate.Every(time.Minute/5)'s refill interval, seconds
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusTooManyRequests)
	_, _ = rw.Write([]byte(`{"errorCode":"RATE_LIMITED","message":"Too many requests. Try again later."}`))
}

// clientIP reads r.RemoteAddr directly — correct only because there is no reverse proxy in front of
// openfaithmap-api today. If one is ever added, this must change to trust a specific, known
// forwarded-for header, never blindly, or the limiter becomes trivially bypassable (Open Seam,
// docs/modules/hardening.md).
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
