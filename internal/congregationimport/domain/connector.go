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
	// token, an Overpass tile index).
	Fetch(ctx context.Context, cursor *string) (batch []RawRecord, nextCursor *string, err error)
	// Normalize maps one connector-native RawRecord onto the common NormalizedCandidate shape.
	Normalize(raw RawRecord) (NormalizedCandidate, error)
}
