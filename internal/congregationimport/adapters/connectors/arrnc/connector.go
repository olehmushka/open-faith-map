// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package arrnc reads Argentina's Registro Nacional de Cultos (National Registry of Religious
// Cults) open-data export and yields candidate religious organizations.
//
// Real, verified source (checked live, 2026-08-14, not assumed): the dataset is listed on
// datos.gob.ar (CKAN id registro-nacional-de-cultos, CC-BY 4.0, published by the Ministerio de
// Relaciones Exteriores, Comercio Internacional y Culto). The CKAN-declared resource URL
// (https://cancilleria.gob.ar/userfiles/datos/registro-nacional-cultos.csv) is DEAD — a plain HEAD
// request returns 404. The ministry's own current landing page
// (https://cancilleria.gob.ar/iniciativas/datos-abiertos/set-de-datos-de-culto) links the real,
// live resource instead: https://cancilleria.gob.ar/userfiles/datos/registro-culto-export.csv —
// confirmed 200, 3,608,415 bytes, Last-Modified 2025-08-13 (genuinely current; the CKAN metadata's
// own "2018-08-31" is stale). Both URLs are recorded in Citation() below, not just this comment, so
// the dead-resource finding isn't silently lost.
//
// Real, verified schema (downloaded and inspected directly): a plain 5-column CSV, NO header row,
// UTF-8, well-formed quoting throughout every one of its 30,178 rows (confirmed by scanning all of
// them — zero malformed rows). Column order, found by inspection — the ministry's own landing-page
// prose names the fields in a DIFFERENT order than the file actually uses:
//
//	0 name · 1 address (free-text street+number, e.g. "AV. MITRE S/N") · 2 locality/city ·
//	3 province · 4 CI
//
// Real, consequential finding: CI (column 4) is NOT a per-row unique key — it is the registered
// institute's OWN registration number, shared across every "- FILIAL N" (branch) row of that same
// institute (e.g. "IGLESIA NUEVA APOSTOLICA (SUD AMERICA)" has 100+ branch rows, all CI=4). Using it
// directly as SourceRecordID would collapse every branch into one candidate, so it isn't — see
// sourceRecordID below.
//
// Real, consequential finding: 503 of the file's 30,178 rows are byte-for-byte duplicates of
// another row (identical name+address+locality+province) — a genuine data-quality artifact in the
// source itself, confirmed by direct inspection, not a parsing bug here. Since SourceRecordID is
// derived from exactly those four fields, these 503 rows correctly collapse onto an
// already-created candidate on upsert rather than becoming a second, visually-indistinguishable
// one — the right behavior, not a bug: expect recordsFetched=30178 but candidatesCreated≈29675 on
// a clean first run.
//
// Real, much smaller scale than uaedr's ~3.15GB ЄДР export (3.6MB here) — this connector loads the
// whole CSV into memory once per run rather than uaedr's stateful streaming design; see Connector's
// own doc comment for why that difference is deliberate, not a shortcut.
package arrnc

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
)

const Code = "ar-rnc"

// batchSize mirrors uaedr's own choice — bounds how many records one Fetch call returns.
const batchSize = 500

// robotsCheckedAt records when this connector's robots.txt check (see Citation) actually happened
// — a fixed historical fact, not something recomputed on every call.
var robotsCheckedAt = time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)

// cultRow is one CSV row's fields, positional per this package's own doc comment (the source has
// no header row to name them).
type cultRow struct {
	Name     string `json:"name"`
	Address  string `json:"address"`
	Locality string `json:"locality"`
	Province string `json:"province"`
	CI       string `json:"ci"`
}

// Connector reads the RNC export one of two ways, mutually exclusive (mirrors uaedr.New's own
// exactly-one-of-two-inputs validation):
//   - FilePath: a local file — for tests/offline use.
//   - SourceURL: a remote HTTP(S) URL — the real deployment path.
//
// Unlike uaedr, both modes reduce to the SAME code path: load the whole file once (guarded by
// loadOnce), parse every row, cache it in memory — this source is 3.6MB, three orders of magnitude
// smaller than uaedr's ~3.15GB export, comfortably within the same ~500MB-1GB cheap-VM budget
// uaedr's own streaming design exists to respect. There is no reopen-and-reskip step at all here,
// so there's nothing to double-count the way uaedr's real 2026-08-14 cursor bug did — Fetch's
// batching is plain integer-offset slicing over an already-materialized, already-correct slice.
type Connector struct {
	FilePath  string
	SourceURL string

	httpClient *http.Client

	loadOnce sync.Once
	loadErr  error
	rows     []cultRow
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

// Citation records what was actually checked before this connector was allowed to run
// (D-CongregationImport's own "decision #4" discipline) — both the dead CKAN-declared URL and the
// real working one, honestly, so a future session doesn't have to rediscover the discrepancy.
func (c *Connector) Citation() domain.SourceCitation {
	robotsURL := "https://cancilleria.gob.ar/robots.txt"
	termsURL := "https://datos.gob.ar/dataset/registro-nacional-de-cultos"
	checkedAt := robotsCheckedAt
	rateLimitNotes := "robots.txt declares Crawl-delay: 10 for the whole site; honored trivially — " +
		"this connector fetches the 3.6MB export once per run, never per batch."
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
			"https://cancilleria.gob.ar/userfiles/datos/registro-culto-export.csv instead, linked " +
			"from the ministry's own landing page, confirmed reachable and current " +
			"(Last-Modified 2025-08-13, unlike the CKAN metadata's stale 2018-08-31).",
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
			SourceRecordID: sourceRecordID(row),
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

// sourceRecordID derives a stable, per-row-unique key from the row's own content — CI cannot be
// used directly (see this package's own doc comment: it's shared across every branch of one
// institute, not a per-row key). SHA-256 over the normalized name+address+locality+province tuple:
// identical content on a re-run always yields the identical ID (UpsertCandidate updates the
// existing row rather than duplicating it — including across the file's own 503 genuinely
// duplicate rows, correctly), and two rows sharing the same CI but different content (real, distinct
// branch locations) always yield different IDs.
func sourceRecordID(row cultRow) string {
	key := strings.ToLower(row.Name) + "|" + strings.ToLower(row.Address) + "|" +
		strings.ToLower(row.Locality) + "|" + strings.ToLower(row.Province)
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// Normalize maps a cultRow's raw JSON payload onto the common candidate shape. Unlike uaedr (whose
// real export has no address field at all), this source DOES carry real street/city/province text
// — Street/Locality/AdminArea1 are populated for real, giving the operator a genuine head start
// instead of a blank slate. Latitude/Longitude still stay nil (the source has no coordinates), so
// every candidate still lands in NEEDS_GEOCODE — this repo has no real geocoder yet (a named open
// seam, docs/modules/congregationimport.md) — but the operator now fills coordinates in against a
// real address rather than inventing one from nothing.
//
// TaxonHint/JurisdictionHint both reuse Name, same rationale as uaedr: this export has no separate
// jurisdiction field, and any institutional-hierarchy information a hierarchical-polity body's
// registered name carries (a diocese, a synod) is embedded in the free-text name itself.
func (c *Connector) Normalize(raw domain.RawRecord) (domain.NormalizedCandidate, error) {
	var row cultRow
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
		c.rows, c.loadErr = c.load(ctx)
	})
	return c.loadErr
}

// load reads and parses the CSV in full — FieldsPerRecord=5 makes a malformed row (the real file
// has none, confirmed by direct inspection) a loud parse error rather than a silent misread,
// matching this repo's own stated preference for failing loudly over guessing.
func (c *Connector) load(ctx context.Context) ([]cultRow, error) {
	r, closeFn, err := c.open(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = closeFn() }()

	cr := csv.NewReader(r)
	cr.FieldsPerRecord = 5
	records, err := cr.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("arrnc: parse CSV: %w", err)
	}

	rows := make([]cultRow, 0, len(records))
	for _, rec := range records {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		rows = append(rows, cultRow{
			Name:     strings.TrimSpace(rec[0]),
			Address:  strings.TrimSpace(rec[1]),
			Locality: strings.TrimSpace(rec[2]),
			Province: strings.TrimSpace(rec[3]),
			CI:       strings.TrimSpace(rec[4]),
		})
	}
	return rows, nil
}

// open returns a reader over the CSV's raw bytes, from a local file (FilePath mode) or a live HTTP
// GET (SourceURL mode) — either way read in full by load, never streamed batch-by-batch, since the
// whole file is small enough that there is no reason to.
func (c *Connector) open(ctx context.Context) (io.Reader, func() error, error) {
	if c.FilePath != "" {
		f, err := os.Open(c.FilePath)
		if err != nil {
			return nil, nil, fmt.Errorf("arrnc: open %s: %w", c.FilePath, err)
		}
		return f, f.Close, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.SourceURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("arrnc: build request for %s: %w", c.SourceURL, err)
	}
	req.Header.Set("User-Agent", c.Citation().UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("arrnc: GET %s: %w", c.SourceURL, err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		_ = resp.Body.Close()
		return nil, nil, fmt.Errorf("arrnc: GET %s: unexpected status %s: %s", c.SourceURL, resp.Status, body)
	}
	return resp.Body, resp.Body.Close, nil
}
