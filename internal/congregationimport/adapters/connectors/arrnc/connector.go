// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package arrnc reads Argentina's Registro Nacional de Cultos (National Registry of Religious
// Cults) open-data export and yields candidate religious organizations.
//
// The CSV fetch/parse itself (including the real dead-CKAN-URL finding, the no-header 5-column
// schema, and the per-row content-hash SourceID that correctly separates institute branches while
// collapsing the source's own genuine duplicate rows) is delegated to
// github.com/olehmushka/go-arrnc, extracted from this connector's own earlier implementation — see
// that package's own doc comment for the full real-world-verified findings. This connector's own
// job is everything go-arrnc deliberately doesn't own: batching over an already-loaded slice and
// mapping onto domain.NormalizedCandidate.
//
// Real, much smaller scale than uaedr's ~3.15GB ЄДР export (3.6MB here) — this connector loads the
// whole CSV into memory once per run rather than uaedr's stateful streaming design; see Connector's
// own doc comment for why that difference is deliberate, not a shortcut.
package arrnc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	upstream "github.com/olehmushka/go-arrnc"
	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
)

const Code = upstream.Code

// batchSize mirrors uaedr's own choice — bounds how many records one Fetch call returns.
const batchSize = 500

// robotsCheckedAt records when this connector's robots.txt check (see Citation) actually happened
// — a fixed historical fact, not something recomputed on every call.
var robotsCheckedAt = time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)

// Connector reads the RNC export one of two ways, mutually exclusive (mirrors uaedr.New's own
// exactly-one-of-two-inputs validation):
//   - FilePath: a local file — for tests/offline use.
//   - SourceURL: a remote HTTP(S) URL — the real deployment path.
//
// Unlike uaedr, both modes reduce to the SAME code path: load the whole file once (guarded by
// loadOnce), parse every row via go-arrnc, cache it in memory — this source is 3.6MB, three orders
// of magnitude smaller than uaedr's ~3.15GB export, comfortably within the same ~500MB-1GB cheap-VM
// budget uaedr's own streaming design exists to respect. There is no reopen-and-reskip step at all
// here, so there's nothing to double-count the way uaedr's real 2026-08-14 cursor bug did — Fetch's
// batching is plain integer-offset slicing over an already-materialized, already-correct slice.
type Connector struct {
	FilePath  string
	SourceURL string

	httpClient *http.Client

	loadOnce sync.Once
	loadErr  error
	rows     []upstream.Row
}

// New constructs an ar-rnc connector. Exactly one of filePath/sourceURL must be set — see
// Connector's own doc comment.
func New(filePath, sourceURL string, httpClient *http.Client) (*Connector, error) {
	if (filePath == "") == (sourceURL == "") {
		return nil, fmt.Errorf("arrnc: exactly one of filePath or sourceURL must be set (got filePath=%q sourceURL=%q)", filePath, sourceURL)
	}
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &Connector{FilePath: filePath, SourceURL: sourceURL, httpClient: httpClient}, nil
}

func (c *Connector) Code() string { return Code }

// Clone returns a fresh Connector for one RunConnector call — same fixed FilePath/SourceURL/
// httpClient, zero loadOnce/rows. Real bug fixed by this (found live 2026-08-14): without Clone,
// a second RunConnector call against the same long-lived, boot-registered instance would silently
// reuse the FIRST run's cached rows forever via loadOnce, never re-fetching from the real source.
func (c *Connector) Clone() domain.Connector {
	return &Connector{FilePath: c.FilePath, SourceURL: c.SourceURL, httpClient: c.httpClient}
}

// Citation records what was actually checked before this connector was allowed to run
// (D-CongregationImport's own "decision #4" discipline) — both the dead CKAN-declared URL and the
// real working one, honestly, so a future session doesn't have to rediscover the discrepancy. The
// URLs/notes themselves are go-arrnc's own documented findings (RobotsTxtURL/TermsURL/
// RateLimitNotes), not re-derived here.
func (c *Connector) Citation() domain.SourceCitation {
	robotsURL := upstream.RobotsTxtURL
	termsURL := upstream.TermsURL
	checkedAt := robotsCheckedAt
	rateLimitNotes := upstream.RateLimitNotes
	return domain.SourceCitation{
		RobotsTxtURL:    &robotsURL,
		RobotsCheckedAt: &checkedAt,
		TermsURL:        &termsURL,
		TermsCheckedAt:  &checkedAt,
		UserAgent:       "openfaithmap-congregationimport/1.0 (structured government open data, not a scrape)",
		RateLimitNotes:  &rateLimitNotes,
		Notes: "Argentina's Ministerio de Relaciones Exteriores, Comercio Internacional y Culto " +
			"publishes the Registro Nacional de Cultos as open data, listed on datos.gob.ar " +
			"(CC-BY 4.0). The CKAN-declared resource URL " +
			"(https://cancilleria.gob.ar/userfiles/datos/registro-nacional-cultos.csv) is dead " +
			"(404, confirmed live) — the real, current export lives at " +
			upstream.DefaultSourceURL + " instead, linked from the ministry's own landing page, " +
			"confirmed reachable and current (Last-Modified 2025-08-13, unlike the CKAN metadata's " +
			"stale 2018-08-31).",
	}
}

// Fetch loads the whole export (once, via ensureLoaded) then serves batchSize-sized batches by
// plain integer-offset slicing — cursor is that offset as a decimal string, nil/empty meaning 0.
func (c *Connector) Fetch(ctx context.Context, cursor *string) (batch []domain.RawRecord, nextCursor *string, err error) {
	if err := c.ensureLoaded(ctx); err != nil {
		return nil, nil, err
	}

	offset := 0
	if cursor != nil && *cursor != "" {
		offset, err = strconv.Atoi(*cursor)
		if err != nil {
			return nil, nil, fmt.Errorf("arrnc: invalid cursor %q: %w", *cursor, err)
		}
	}
	if offset >= len(c.rows) {
		return nil, nil, nil
	}

	end := offset + batchSize
	if end > len(c.rows) {
		end = len(c.rows)
	}

	now := time.Now()
	batch = make([]domain.RawRecord, 0, end-offset)
	for _, row := range c.rows[offset:end] {
		payload, merr := json.Marshal(row)
		if merr != nil {
			return nil, nil, fmt.Errorf("arrnc: marshal raw payload: %w", merr)
		}
		batch = append(batch, domain.RawRecord{
			SourceRecordID: row.SourceID(),
			RawPayload:     payload,
			FetchedAt:      now,
		})
	}

	if end >= len(c.rows) {
		return batch, nil, nil // source exhausted
	}
	next := strconv.Itoa(end)
	return batch, &next, nil
}

// Normalize maps a Row's raw JSON payload onto the common candidate shape. Unlike uaedr (whose
// real export has no address field at all), this source DOES carry real street/city/province text
// — Street/Locality/AdminArea1 are populated for real, giving the operator a genuine head start
// instead of a blank slate. Latitude/Longitude still stay nil (the source has no coordinates) —
// congregationimport's own real geocoder (application.Geocoder/nominatim) is advisory-only, not
// wired into Normalize, so every candidate still lands in NEEDS_GEOCODE here, same as before.
//
// TaxonHint/JurisdictionHint both reuse Name, same rationale as uaedr: this export has no separate
// jurisdiction field, and any institutional-hierarchy information a hierarchical-polity body's
// registered name carries (a diocese, a synod) is embedded in the free-text name itself.
func (c *Connector) Normalize(raw domain.RawRecord) (domain.NormalizedCandidate, error) {
	var row upstream.Row
	if err := json.Unmarshal(raw.RawPayload, &row); err != nil {
		return domain.NormalizedCandidate{}, fmt.Errorf("arrnc: unmarshal raw payload: %w", err)
	}

	hint := row.Name
	country := "Argentina"
	nc := domain.NormalizedCandidate{
		Name:             row.Name,
		TaxonHint:        &hint,
		JurisdictionHint: &hint,
		CountryHint:      &country,
	}
	if row.Address != "" {
		addr := row.Address
		nc.Street = &addr
	}
	if row.Locality != "" {
		locality := row.Locality
		nc.Locality = &locality
	}
	if row.Province != "" {
		province := row.Province
		nc.AdminArea1 = &province
	}
	return nc, nil
}

// ensureLoaded fetches and parses the whole export exactly once per Connector instance, regardless
// of how many Fetch calls follow — loadOnce makes this safe under concurrent Fetch calls too,
// though RunConnector itself never issues concurrent Fetch calls on one connector.
func (c *Connector) ensureLoaded(ctx context.Context) error {
	c.loadOnce.Do(func() {
		if c.FilePath != "" {
			c.rows, c.loadErr = upstream.FetchFile(c.FilePath)
			return
		}
		c.rows, c.loadErr = upstream.Fetch(ctx, c.SourceURL, upstream.FetchOptions{
			HTTPClient: c.httpClient,
			UserAgent:  c.Citation().UserAgent,
		})
	})
	return c.loadErr
}
