// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"context"
	"time"
)

// SourceCitation documents what was checked before a connector was allowed to run — the decision
// #4 discipline ("check robots.txt/ToS per HTML source, don't assume") made durable and queryable
// (congregationimport_connector_citations) rather than left in a code comment.
type SourceCitation struct {
	RobotsTxtURL    *string
	RobotsCheckedAt *time.Time
	TermsURL        *string
	TermsCheckedAt  *time.Time
	UserAgent       string
	RateLimitNotes  *string
	Notes           string
}

// Connector is the Strategy-pattern interface every source (bulk open-data dump, Overpass API,
// HTML scrape) implements identically, so application.Service.RunConnector never branches on
// source type.
type Connector interface {
	// Code is the stable source_code stored on every run/candidate row.
	Code() string
	// Citation is this connector's own SourceCitation. Bulk/API connectors may return a zero value;
	// HTML connectors must return a real one (adapters/connectors/html/base enforces this).
	Citation() SourceCitation
	// Fetch returns one batch of raw records plus a cursor to resume from, or a nil cursor when the
	// source is exhausted. Cursor semantics are connector-defined (e.g. a byte offset, a page
	// token, an Overpass tile index). An HTTP-streaming source may treat cursor as an opaque
	// "has this run already started" marker backed by internal, run-scoped stream state (an open
	// response body + decoder held across calls) rather than a literal reopen-from-scratch offset —
	// re-fetching a multi-hundred-MB source from byte zero on every batch is not viable. A connector
	// doing this must guard against two concurrent Fetch sequences racing on the same instance (see
	// ConnectorCloser) and should document that a mid-run crash cannot cheaply resume mid-stream.
	Fetch(ctx context.Context, cursor *string) (batch []RawRecord, nextCursor *string, err error)
	// Normalize maps one connector-native RawRecord onto the common NormalizedCandidate shape.
	Normalize(raw RawRecord) (NormalizedCandidate, error)
	// Clone returns a fresh Connector value carrying the same fixed deployment config (file path,
	// source URL, HTTP client) but zero run-scoped mutable state (in-memory caches, open streams,
	// locks). RunConnector calls this (or WithParameters, for a ConnectorConfigurable connector when
	// the operator supplies run parameters) exactly once per run, so a long-lived, boot-registered
	// instance's state from a PRIOR run can never leak into this one — real bug found live
	// 2026-08-14: arrnc/osm's sync.Once-cached in-memory rows meant a second RunConnector call
	// against the same registered instance silently replayed the first run's data forever, never
	// re-querying the real source. No revalidation is needed here — the original instance was
	// already validated at construction.
	Clone() Connector
}

// ConnectorCloser is an optional interface a Connector implements when it holds run-scoped
// resources across Fetch calls (e.g. an open HTTP response body and a run semaphore) that must be
// released even if a run ends via ctx cancellation or a Fetch error, not just clean source
// exhaustion — RunConnector's own loop only calls Fetch again on the success path, so without this
// hook such a connector would leak its lock/stream forever on any error exit. RunConnector calls
// Close via a deferred, best-effort call whenever a connector implements it.
type ConnectorCloser interface {
	Close() error
}

// ConnectorConfigurable is implemented by a connector with genuine run-time parameters an operator
// might reasonably vary per RunConnector call (e.g. osm's CountryCodes — "which subset of data
// should this run cover"), as distinct from fixed deployment config (a file path, a base URL)
// nobody would reasonably override per click of a button. WithParameters returns a fresh Connector
// value scoped to this one run (same freshness guarantee as Clone, and callers should never call
// both) with the given overrides applied — never mutates the receiver. An unrecognized parameter
// key or an invalid value must be a clear, wrapped error, never a silent no-op or a silently
// ignored key.
type ConnectorConfigurable interface {
	WithParameters(params map[string]string) (Connector, error)
}
