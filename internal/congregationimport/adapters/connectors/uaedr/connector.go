// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package uaedr reads Ukraine's ЄДР (Unified State Register of Legal Entities) open-data export
// and yields candidate religious organizations. Real, verified source: the Ministry of Justice
// publishes this as genuine open data at data.gov.ua (dataset id
// 03cc1239-3988-4451-aa0d-aadb77448714, resource "uo.zip" — legal entities), updated weekly, free.
//
// Real, verified schema (checked against the dataset's own published uo_schema.zip, an XSD, not
// inferred): the export is a flat stream of <SUBJECT> elements, each carrying (among many fields
// this connector ignores) NAME, SHORT_NAME, OPF (organizational-legal form — "Організаційно-правова
// форма"), EDRPOU (the unique code, this connector's SourceRecordID), and STAN (activity status).
// Religious organizations are identified by OPF = "Релігійна організація" (КОПФГ classifier code
// 825, verified against data.gov.ua's own kopfg.json resource) — matched case-insensitively
// (strings.EqualFold), a real correction found by scanning the actual downloaded export: the live
// data stores OPF as "РЕЛІГІЙНА ОРГАНІЗАЦІЯ" (all uppercase), not the classifier's own title-case
// text. An exact match would have matched zero real rows.
//
// Real, verified constraint: this export has NO address field at all (an older, now-superseded
// schema had one; the current one does not). Every candidate this connector produces has
// Latitude/Longitude and every address field nil — congregationimport's pipeline routes that
// straight to NEEDS_GEOCODE, and an operator fills location in manually during review. This is
// documented here rather than silently worked around, since inventing an address field that
// doesn't exist in the real source would be worse than admitting the gap.
//
// Real, verified encoding: windows-1251 (checked directly against the actual downloaded export's
// XML prolog — `<?xml version="1.0" encoding="windows-1251"?>` — not assumed; the schema XSD's own
// "UTF-8" declaration describes the XSD file itself, not the data file). This connector's
// charsetReader auto-detects from the prolog rather than hardcoding either encoding.
//
// Not yet verified: the exact STAN values distinguishing an active organization from a terminated
// one (this connector does not filter on STAN in v1 — every OPF-matched record is staged
// regardless, with the raw STAN value preserved in RawPayload for an operator to see) — flagged as
// an open follow-up, not silently assumed.
package uaedr

import (
	"archive/zip"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
	"golang.org/x/text/encoding/charmap"
)

const Code = "ua-edr"

// religiousOrgOPF is КОПФГ code 825's literal text — verified directly against data.gov.ua's own
// kopfg.json classifier resource, not guessed.
const religiousOrgOPF = "Релігійна організація"

// batchSize bounds how many <SUBJECT> records one Fetch call reads before returning — keeps memory
// bounded regardless of the source file's real size (hundreds of thousands of records).
const batchSize = 500

// subject mirrors the real fields of a <SUBJECT> element this connector actually uses, per
// uo_schema.zip's published XSD. Every other real field (FOUNDERS, SIGNERS, BRANCHES, ...) is
// intentionally not modeled — this connector only needs enough to identify and name a candidate,
// not to reconstruct the full legal-entity record.
type subject struct {
	Record string `xml:"RECORD" json:"record"`
	Name   string `xml:"NAME" json:"name"`
	Short  string `xml:"SHORT_NAME" json:"shortName"`
	OPF    string `xml:"OPF" json:"opf"`
	EDRPOU string `xml:"EDRPOU" json:"edrpou"`
	Stan   string `xml:"STAN" json:"stan"`
}

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

// fetchFile reads batchSize <SUBJECT> records starting after skipping the number of records named
// by cursor (a decimal string; nil/empty means start from the beginning), filters to OPF ==
// religiousOrgOPF, and returns the matched raw records plus a cursor to resume from. Streams via
// xml.Decoder — never loads the full file into memory, since a real export is hundreds of
// thousands of records.
//
// seen counts every <SUBJECT> element visited in THIS reopened pass, starting from 1 — which
// already includes the skip re-decode, since the pass walks the file from byte zero every time. So
// by the time this call returns, seen already equals the correct new cumulative position (skip +
// however many new elements this call actually processed) — the cursor returned must be seen
// alone. A real bug, found live (2026-08-14) against the real ~2M-record export via HTTP-streaming
// mode's independently-correct count (30,721 OPF matches, cross-checked against a plain
// unzip+grep) disagreeing 10x with this path's own count on the same file: this used to return
// cursorOf(seen), double-counting the skip prefix on every call after the first (skip=0 on
// the first call hid it completely). The error compounds across calls — each returned cursor
// carries an extra, growing copy of the previous skip — until the inflated value races past the
// file's true record count and dec.Token() hits a real io.EOF far too early, which looks
// indistinguishable from a clean, complete run (SUCCEEDED, not FAILED) unless independently
// cross-checked, which is exactly how this was caught. Every prior "full-scale" run of this
// connector needing more than one batch (over 500 OPF matches) undercounted as a result — see
// docs/milestones.md's corrected M8 entry.
func (c *Connector) fetchFile(ctx context.Context, cursor *string) (batch []domain.RawRecord, nextCursor *string, err error) {
	skip := 0
	if cursor != nil && *cursor != "" {
		skip, err = strconv.Atoi(*cursor)
		if err != nil {
			return nil, nil, fmt.Errorf("uaedr: invalid cursor %q: %w", *cursor, err)
		}
	}

	r, closeFn, err := c.openXML()
	if err != nil {
		return nil, nil, err
	}
	// A close failure on a read-only stream carries no data-integrity consequence worth surfacing
	// over whatever this Fetch call itself returns — logged via the named err return only if
	// nothing else already failed, never silently dropped.
	defer func() {
		if closeErr := closeFn(); closeErr != nil && err == nil {
			err = fmt.Errorf("uaedr: close: %w", closeErr)
		}
	}()

	dec := xml.NewDecoder(r)
	dec.CharsetReader = charsetReader

	seen := 0
	for {
		if err := ctx.Err(); err != nil {
			return batch, cursorOf(seen), err
		}
		tok, err := dec.Token()
		if err == io.EOF {
			return batch, nil, nil // source exhausted — nextCursor nil signals "done"
		}
		if err != nil {
			return batch, cursorOf(seen), fmt.Errorf("uaedr: decode: %w", err)
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "SUBJECT" {
			continue
		}

		var s subject
		if err := dec.DecodeElement(&s, &start); err != nil {
			return batch, cursorOf(seen), fmt.Errorf("uaedr: decode SUBJECT: %w", err)
		}
		seen++
		if seen <= skip {
			continue // already yielded in a prior Fetch call
		}
		if !strings.EqualFold(strings.TrimSpace(s.OPF), religiousOrgOPF) {
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

// Normalize maps a subject's raw JSON payload onto the common candidate shape. No address fields
// are set — see this package's own doc comment for why. TaxonHint carries the full legal name; the
// application layer's taxon-alias matching does the denomination-keyword resolution, not this
// connector (Fetch/Normalize stay source-shaped, never taxonomy-aware).
//
// JurisdictionHint reuses the same NAME string, not a separate field — re-verified against
// uo_schema.zip's SUBJECT element (this package's own doc comment): there is no dedicated
// parent-organization/jurisdiction field in this export (FOUNDERS/SIGNERS/BRANCHES, the only other
// modeled-adjacent fields, don't carry it either). What the real data does carry, for
// hierarchical-polity bodies, is the eparchy/diocese/deanery named directly inside the legal NAME
// itself (e.g. a UGCC parish's registered name routinely reads "...ПАРАФІЯ ... ЛЬВІВСЬКОЇ
// АРХІЄПАРХІЇ УКРАЇНСЬКОЇ ГРЕКО-КАТОЛИЦЬКОЇ ЦЕРКВИ" — the archeparchy is textually present).
// Independent-polity registrations (most Protestant congregations) simply won't contain any
// jurisdiction-alias substring, and correctly produce no suggestion (application.matchJurisdiction).
func (c *Connector) Normalize(raw domain.RawRecord) (domain.NormalizedCandidate, error) {
	var s subject
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

func (c *Connector) openXML() (io.Reader, func() error, error) {
	f, err := os.Open(c.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("uaedr: open %s: %w", c.FilePath, err)
	}
	if !strings.HasSuffix(strings.ToLower(c.FilePath), ".zip") {
		return f, f.Close, nil
	}

	// f itself must stay open for the whole streaming read below — zip.File.Open()'s returned
	// reader reads lazily from f's underlying ReaderAt, not eagerly into memory. A bare
	// `defer f.Close()` here would close it before the caller ever streams through the returned
	// reader, breaking every read after this function returns (a real bug, caught before it was
	// ever run against a real file: f.Close() closing the very handle the returned reader still
	// needs). The returned close function closes both, in order.
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, nil, fmt.Errorf("uaedr: stat %s: %w", c.FilePath, err)
	}
	zr, err := zip.NewReader(f, info.Size())
	if err != nil {
		_ = f.Close()
		return nil, nil, fmt.Errorf("uaedr: open zip %s: %w", c.FilePath, err)
	}
	for _, zf := range zr.File {
		if strings.HasSuffix(strings.ToLower(zf.Name), ".xml") {
			rc, err := zf.Open()
			if err != nil {
				_ = f.Close()
				return nil, nil, fmt.Errorf("uaedr: open %s in zip: %w", zf.Name, err)
			}
			closeBoth := func() error {
				rcErr := rc.Close()
				fErr := f.Close()
				if rcErr != nil {
					return rcErr
				}
				return fErr
			}
			return rc, closeBoth, nil
		}
	}
	_ = f.Close()
	return nil, nil, fmt.Errorf("uaedr: no .xml entry found in %s", c.FilePath)
}

func cursorOf(n int) *string {
	s := strconv.Itoa(n)
	return &s
}

// charsetReader lets xml.Decoder handle a non-UTF-8 prolog declaration (e.g. a cp1251-encoded
// export, the older schema version's real encoding) without failing outright — Go's stdlib xml
// package only understands UTF-8/US-ASCII natively.
func charsetReader(charset string, input io.Reader) (io.Reader, error) {
	switch strings.ToLower(charset) {
	case "utf-8", "us-ascii", "":
		return input, nil
	case "windows-1251", "cp1251":
		return charmap.Windows1251.NewDecoder().Reader(input), nil
	default:
		return nil, fmt.Errorf("uaedr: unsupported XML charset %q", charset)
	}
}
