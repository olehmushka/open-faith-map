// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package uaedr reads Ukraine's ЄДР (Unified State Register of Legal Entities) open-data export
// and yields candidate religious organizations. Real, verified source: the Ministry of Justice
// publishes this as genuine open data at data.gov.ua (dataset id
// 03cc1239-3988-4451-aa0d-aadb77448714, resource "uo.zip" — legal entities), updated weekly, free.
//
// The wire-format parsing (XML schema, charset auto-detection, OPF religious-org matching, and the
// hand-rolled streaming zip reader an HTTP source needs) is delegated to
// github.com/olehmushka/go-uaedr, extracted from this connector's own earlier implementation — see
// that package's own doc comment for the full real-world-verified schema/encoding/OPF findings.
// This connector's own job is everything go-uaedr deliberately doesn't own: batching, the
// reopen-and-reskip vs. held-open-stream cursor strategies, and mapping onto
// domain.NormalizedCandidate.
//
// Real, verified constraint: this export has NO address field at all (an older, now-superseded
// schema had one; the current one does not). Every candidate this connector produces has
// Latitude/Longitude and every address field nil — congregationimport's pipeline routes that
// straight to NEEDS_GEOCODE, and an operator fills location in manually during review. This is
// documented here rather than silently worked around, since inventing an address field that
// doesn't exist in the real source would be worse than admitting the gap.
package uaedr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	upstream "github.com/olehmushka/go-uaedr"
	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
)

const Code = upstream.Code

// batchSize bounds how many records one Fetch call reads before returning — keeps memory bounded
// regardless of the source file's real size (hundreds of thousands of records).
const batchSize = 500

// Connector reads ЄДР's uo.zip/uo.xml one of two ways, mutually exclusive:
//   - FilePath: a local file (docker-compose-mounted, this repo's own scripts/bootstrap-*
//     precedent for "operator supplies a local file", matching hermenea's own file-shaped
//     connector-type) — Fetch is stateless, reopening and reskipping from scratch every call.
//   - SourceURL: a remote HTTP(S) URL — Fetch is stateful (connector_http.go), holding one open
//     stream across calls, for a deployment that can't or won't stage the ~326MB export on local
//     disk (a cheap, memory-constrained cloud VM). Cursor semantics differ accordingly; see
//     domain.Connector's own doc comment.
type Connector struct {
	FilePath  string
	SourceURL string

	httpClient *http.Client
	http       httpModeState
}

// New constructs a ua-edr connector. Exactly one of filePath/sourceURL must be set — they imply
// genuinely different Fetch state machines (see Connector's doc comment), not just a different
// input location. httpClient is only used in SourceURL mode; nil defaults to an explicit
// no-Timeout *http.Client — this is a multi-hundred-MB streaming download, so a fixed deadline
// would fail slow connections arbitrarily. ctx passed to Fetch is the only cancellation mechanism.
func New(filePath, sourceURL string, httpClient *http.Client) (*Connector, error) {
	if (filePath == "") == (sourceURL == "") {
		return nil, fmt.Errorf("uaedr: exactly one of filePath or sourceURL must be set (got filePath=%q sourceURL=%q)", filePath, sourceURL)
	}
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &Connector{FilePath: filePath, SourceURL: sourceURL, httpClient: httpClient}, nil
}

func (c *Connector) Code() string { return Code }

// Clone returns a fresh Connector for one RunConnector call — same fixed FilePath/SourceURL/
// httpClient, zero httpModeState (a fresh run must open its own stream/lock, never inherit a prior
// run's). FilePath mode already re-reads from scratch on every Fetch(nil, ...) call regardless, so
// this matters most for SourceURL mode, but Clone always returns a pristine instance either way —
// one uniform contract every connector honors, not a per-mode special case.
func (c *Connector) Clone() domain.Connector {
	return &Connector{FilePath: c.FilePath, SourceURL: c.SourceURL, httpClient: c.httpClient}
}

func (c *Connector) Citation() domain.SourceCitation {
	return domain.SourceCitation{
		UserAgent: "openfaithmap-congregationimport/1.0 (structured government open data, not a scrape)",
		Notes: "Ukraine's Ministry of Justice publishes the ЄДР (Unified State Register of Legal " +
			"Entities) as open data at data.gov.ua (dataset 03cc1239-3988-4451-aa0d-aadb77448714, " +
			"resource uo.zip), weekly, free — a licensed bulk download, not a scrape. No robots.txt " +
			"or ToS check applies the way it does to an HTML connector.",
	}
}

// Fetch dispatches to fetchFile (FilePath mode: stateless, reopen-and-reskip) or fetchHTTP
// (SourceURL mode: stateful, one held-open stream across calls) — see Connector's own doc comment
// for why these are genuinely different state machines, not just a different input location.
func (c *Connector) Fetch(ctx context.Context, cursor *string) (batch []domain.RawRecord, nextCursor *string, err error) {
	if c.SourceURL != "" {
		return c.fetchHTTP(ctx, cursor)
	}
	return c.fetchFile(ctx, cursor)
}

// fetchFile reads batchSize religious-org records starting after skipping the number of records
// named by cursor (a decimal string; nil/empty means start from the beginning), and returns the
// matched raw records plus a cursor to resume from. Reopens go-uaedr's own streaming Reader from
// scratch on every call — never loads the full file into memory, since a real export is hundreds of
// thousands of records.
//
// seen counts every record visited in THIS reopened pass, starting from 1 — which already includes
// the skip re-decode, since the pass walks the file from the beginning every time. So by the time
// this call returns, seen already equals the correct new cumulative position (skip + however many
// new elements this call actually processed) — the cursor returned must be seen alone, not
// cursorOf(skip + seen). See docs/milestones-2026-08-07-2026-08-26.md's corrected M8 entry for the real double-counting
// bug this used to have before go-uaedr's Reader existed as a separate package.
func (c *Connector) fetchFile(ctx context.Context, cursor *string) (batch []domain.RawRecord, nextCursor *string, err error) {
	skip := 0
	if cursor != nil && *cursor != "" {
		skip, err = strconv.Atoi(*cursor)
		if err != nil {
			return nil, nil, fmt.Errorf("uaedr: invalid cursor %q: %w", *cursor, err)
		}
	}

	r, closer, err := upstream.OpenFile(c.FilePath)
	if err != nil {
		return nil, nil, err
	}
	// A close failure on a read-only stream carries no data-integrity consequence worth surfacing
	// over whatever this Fetch call itself returns — logged via the named err return only if
	// nothing else already failed, never silently dropped.
	defer func() {
		if closeErr := closer.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("uaedr: close: %w", closeErr)
		}
	}()

	seen := 0
	for {
		if err := ctx.Err(); err != nil {
			return batch, cursorOf(seen), err
		}
		s, err := r.Next()
		if err == io.EOF {
			return batch, nil, nil // source exhausted — nextCursor nil signals "done"
		}
		if err != nil {
			return batch, cursorOf(seen), fmt.Errorf("uaedr: decode: %w", err)
		}
		seen++
		if seen <= skip {
			continue // already yielded in a prior Fetch call
		}
		if !s.IsReligiousOrg() {
			continue
		}

		payload, err := json.Marshal(s)
		if err != nil {
			return batch, cursorOf(seen), fmt.Errorf("uaedr: marshal raw payload: %w", err)
		}
		batch = append(batch, domain.RawRecord{
			SourceRecordID: s.EDRPOU,
			RawPayload:     payload,
			FetchedAt:      time.Now(),
		})
		if len(batch) >= batchSize {
			return batch, cursorOf(seen), nil
		}
	}
}

// Normalize maps a Subject's raw JSON payload onto the common candidate shape. No address fields
// are set — see this package's own doc comment for why. TaxonHint carries the full legal name; the
// application layer's taxon-alias matching does the denomination-keyword resolution, not this
// connector (Fetch/Normalize stay source-shaped, never taxonomy-aware).
//
// JurisdictionHint reuses the same NAME string, not a separate field — go-uaedr's own Subject
// carries no dedicated parent-organization/jurisdiction field (FOUNDERS/SIGNERS/BRANCHES, the only
// other modeled-adjacent fields, don't carry it either). What the real data does carry, for
// hierarchical-polity bodies, is the eparchy/diocese/deanery named directly inside the legal NAME
// itself (e.g. a UGCC parish's registered name routinely reads "...ПАРАФІЯ ... ЛЬВІВСЬКОЇ
// АРХІЄПАРХІЇ УКРАЇНСЬКОЇ ГРЕКО-КАТОЛИЦЬКОЇ ЦЕРКВИ" — the archeparchy is textually present).
// Independent-polity registrations (most Protestant congregations) simply won't contain any
// jurisdiction-alias substring, and correctly produce no suggestion (application.matchJurisdiction).
func (c *Connector) Normalize(raw domain.RawRecord) (domain.NormalizedCandidate, error) {
	var s upstream.Subject
	if err := json.Unmarshal(raw.RawPayload, &s); err != nil {
		return domain.NormalizedCandidate{}, fmt.Errorf("uaedr: unmarshal raw payload: %w", err)
	}
	name := strings.TrimSpace(s.Name)
	if name == "" {
		name = strings.TrimSpace(s.Short)
	}
	hint := name
	return domain.NormalizedCandidate{
		Name:             name,
		TaxonHint:        &hint,
		JurisdictionHint: &hint,
	}, nil
}

func cursorOf(n int) *string {
	s := strconv.Itoa(n)
	return &s
}
