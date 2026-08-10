// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/palantir/pkg/metrics"
	"github.com/palantir/witchcraft-go-server/v2/wrouter"
	"golang.org/x/time/rate"
)

// rateLimitKey is (client IP, endpoint) — two independent buckets per caller so one endpoint's
// traffic can't starve the other's budget (D-Hardening, docs/architecture/decisions.md).
type rateLimitKey struct {
	ip       string
	endpoint string
}

// RateLimiter is an in-process, per-(client IP, endpoint) token-bucket limiter, scoped to exactly
// ModerationPublicService's two anonymous write endpoints (M7). State is single-process and
// ephemeral by design — a restart clears every bucket, and it does not coordinate across replicas;
// fine while openfaithmap-api runs single-replica (docs/modules/hardening.md's Open Seams).
//
// The limiters map never evicts entries — a long-running process accumulates one *rate.Limiter per
// distinct (IP, endpoint) pair forever. Known, accepted limitation, not solved here: there is no
// data on real request volume to size an eviction policy against yet.
type RateLimiter struct {
	mu       sync.Mutex
	limiters map[rateLimitKey]*rate.Limiter
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{limiters: make(map[rateLimitKey]*rate.Limiter)}
}

func (rl *RateLimiter) allow(ip, endpoint string) bool {
	key := rateLimitKey{ip: ip, endpoint: endpoint}
	rl.mu.Lock()
	lim, ok := rl.limiters[key]
	if !ok {
		// Provisional, not data-tuned (D-Hardening): ~5 requests/minute sustained, burst of 5.
		lim = rate.NewLimiter(rate.Every(time.Minute/5), 5)
		rl.limiters[key] = lim
	}
	rl.mu.Unlock()
	return lim.Allow()
}

// Middleware implements wrouter.RouteHandlerMiddleware. Runs pre-auth, before the request reaches
// the generated Conjure handler — on reject, writes a raw 429 directly, never a Conjure-typed error
// (Conjure's fixed error-code system has no code that maps to HTTP 429; this is a deliberate,
// permanent departure from this repo's Conjure-error-body convention, not a gap to fix later).
func (rl *RateLimiter) Middleware(rw http.ResponseWriter, r *http.Request, reqVals wrouter.RequestVals, next wrouter.RouteRequestHandler) {
	ip := clientIP(r)
	if rl.allow(ip, r.URL.Path) {
		next(rw, r, reqVals)
		return
	}
	metrics.FromContext(r.Context()).Counter("openfaithmap.moderation.rate_limit_rejections").Inc(1)
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
