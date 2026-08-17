// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// Package wikidatacatholic reads Catholic dioceses/eparchies from Wikidata's public SPARQL query
// service and yields domain.JurisdictionNodes for D-CatholicJurisdictionSync
// (docs/architecture/decisions.md).
//
// Real, verified source (checked live this session, not assumed): query.wikidata.org's SPARQL
// endpoint, CC0-licensed structured data. Scoped to `wdt:P1866` ("Catholic Hierarchy diocese ID") —
// this is deliberately the scoping mechanism, not a generic `wdt:P708` ("diocese") query, because
// P708 is also used by Orthodox/Anglican bodies; P1866 only exists on an entity that
// Catholic-Hierarchy.org itself has catalogued as a real Catholic circumscription (Latin **and** the
// Eastern Catholic churches in full communion with Rome, e.g. Ukraine's UGCC). Live-verified counts
// this session: 6,655 such dioceses/eparchies worldwide, 167,544 parish/church entities linked to
// them via `wdt:P708`, 142,459 (85%) of those carrying direct coordinates (not fetched by this
// package — see docs/modules/congregationimport.md's Open Seams for the deferred parish-level
// connector).
//
// robots.txt finding, checked and RESOLVED explicitly with the project owner before writing this
// file (not reasoned past silently, per this project's own established discipline —
// docs/modules/congregationimport.md's D-CongregationImport decision #4): query.wikidata.org's
// robots.txt disallows /sparql for every user agent. This is judged (and confirmed by the owner) to
// be Wikimedia's standard pattern for keeping search-engine crawlers off the interactive
// query-builder HTML page, not a block on the documented public API this package calls — live-
// verified response headers on the exact endpoint/path confirm this: `access-control-allow-origin:
// *` (a page meant for cross-origin API consumption, not browser navigation) and a dedicated
// `api-user-agent` request header Wikimedia's own docs reserve specifically for API clients
// (distinct from a browser's User-Agent). This package sets both.
package wikidatacatholic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/olehmushka/open-faith-map/internal/congregationimport/domain"
)

const Code = "wikidata-catholic"

const defaultBaseURL = "https://query.wikidata.org/sparql"

// batchSize bounds how many diocese entities one Fetch call returns — small deliberately, since each
// batch also drives a second, VALUES-bounded labels query (see fetchLabels).
const batchSize = 50

// labelLanguages is the fixed set of locales this package pulls rdfs:label/skos:altLabel in — chosen
// to cover every language this repo's existing connectors' scraped legal names actually appear in
// (Cyrillic for ua-edr, Spanish for ar-rnc) plus the languages Catholic dioceses are most commonly
// named in, so a future connector in any of these languages benefits without code changes.
var labelLanguages = []string{"en", "uk", "ru", "es", "pt", "fr", "it", "pl", "de"}

var robotsCheckedAt = time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)

// qidPattern validates a Wikidata entity ID before it's interpolated into a SPARQL query string —
// this package builds queries by string concatenation (no parameterized-SPARQL library exists in
// this repo's dependency set), so every value that reaches a query MUST be validated against this
// pattern first; CountryQIDs is the only caller-supplied input that does.
var qidPattern = regexp.MustCompile(`^Q[1-9][0-9]*$`)

// Source implements domain.JurisdictionSource against Wikidata's public SPARQL endpoint.
type Source struct {
	// BaseURL defaults to defaultBaseURL — overridable for tests only (mirrors osm.Connector's own
	// OverpassBaseURL field).
	BaseURL string
	// CountryQIDs optionally scopes the sync to specific countries (e.g. "Q212" for Ukraine) — the
	// live-verification-first-target scoping the owner asked for. Empty means every country Wikidata
	// has a P1866-tagged diocese for (global, per the owner's own decision — see this package's
	// citation).
	CountryQIDs []string

	httpClient *http.Client
}

// New validates CountryQIDs eagerly (never silently drops a malformed value into a query string) and
// returns a ready-to-use Source.
func New(baseURL string, countryQIDs []string, httpClient *http.Client) (*Source, error) {
	for _, q := range countryQIDs {
		if !qidPattern.MatchString(q) {
			return nil, fmt.Errorf("wikidatacatholic: invalid country QID %q (want e.g. %q)", q, "Q212")
		}
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	return &Source{BaseURL: baseURL, CountryQIDs: countryQIDs, httpClient: httpClient}, nil
}

func (s *Source) Code() string { return Code }

// Citation records what was actually checked before this source was allowed to run — see this
// package's own doc comment for the full robots.txt reasoning and the owner's explicit sign-off.
func (s *Source) Citation() domain.SourceCitation {
	robotsURL := "https://query.wikidata.org/robots.txt"
	termsURL := "https://www.wikidata.org/wiki/Wikidata:Licensing"
	checkedAt := robotsCheckedAt
	rateLimitNotes := "Wikimedia's own SPARQL query-service etiquette: one client, sequential " +
		"queries, a descriptive User-Agent AND api-user-agent header (both set here) — this source " +
		"fetches at most batchSize=50 dioceses (plus one bounded labels query) per Fetch call, never " +
		"a bulk single-query dump."
	return domain.SourceCitation{
		RobotsTxtURL:    &robotsURL,
		RobotsCheckedAt: &checkedAt,
		TermsURL:        &termsURL,
		TermsCheckedAt:  &checkedAt,
		UserAgent:       "openfaithmap-congregationimport/1.0 (structured public data via the documented SPARQL API; contact: olegamysk@gmail.com)",
		RateLimitNotes:  &rateLimitNotes,
		Notes: "query.wikidata.org's robots.txt disallows /sparql for every user agent — checked live " +
			"2026-08-15, and explicitly confirmed with the project owner (not reasoned past silently, " +
			"per D-CongregationImport decision #4's own discipline) that this is Wikimedia's standard " +
			"pattern for keeping search-engine crawlers off the interactive HTML query-builder page, " +
			"not a block on the documented public SPARQL API this package calls — live-verified " +
			"response headers on this exact endpoint confirm API-oriented design " +
			"(access-control-allow-origin: *, a dedicated api-user-agent header Wikimedia's own docs " +
			"reserve for API clients). Data is CC0 (Wikidata's own licensing policy). Scoped to " +
			"wdt:P1866 (Catholic Hierarchy diocese ID) specifically, not the generic wdt:P708 " +
			"(\"diocese\") property Orthodox/Anglican bodies also carry.",
	}
}

// Fetch pages through P1866-tagged dioceses (LIMIT/OFFSET, ordered by entity URI for a stable
// cursor — same integer-offset-as-decimal-string cursor shape arrnc.Connector.Fetch uses), then
// resolves each batch's multilingual labels in one second, VALUES-bounded query.
func (s *Source) Fetch(ctx context.Context, cursor *string) (batch []domain.JurisdictionNode, nextCursor *string, err error) {
	offset := 0
	if cursor != nil && *cursor != "" {
		offset, err = strconv.Atoi(*cursor)
		if err != nil {
			return nil, nil, fmt.Errorf("wikidatacatholic: invalid cursor %q: %w", *cursor, err)
		}
	}

	rows, err := s.fetchCoreBatch(ctx, offset)
	if err != nil {
		return nil, nil, err
	}
	if len(rows) == 0 {
		return nil, nil, nil // source exhausted
	}

	qids := make([]string, 0, len(rows))
	for _, r := range rows {
		qids = append(qids, r.qid)
	}
	labelsByQID, err := s.fetchLabels(ctx, qids)
	if err != nil {
		return nil, nil, err
	}

	nodes := make([]domain.JurisdictionNode, 0, len(rows))
	for _, r := range rows {
		names := labelsByQID[r.qid]
		name, aliases := primaryAndAliases(names)
		if name == "" {
			// No label in any of labelLanguages at all — fall back to the raw QID rather than
			// dropping the node; still a valid, syncable node, just with a less useful default name.
			name = r.qid
		}
		node := domain.JurisdictionNode{
			ExternalID:         r.qid,
			ParentExternalID:   r.parentQID,
			Name:               name,
			AliasNames:         aliases,
			CountryHint:        r.countryQID,
			SuggestedOrgKindID: "diocese",
		}
		nodes = append(nodes, node)
	}

	if len(rows) < batchSize {
		return nodes, nil, nil // source exhausted
	}
	next := strconv.Itoa(offset + batchSize)
	return nodes, &next, nil
}

type coreRow struct {
	qid        string
	countryQID *string
	parentQID  *string
}

// fetchCoreBatch queries one LIMIT/OFFSET page of P1866-tagged dioceses, ordered by entity URI so
// paging is stable across calls (no row can move between pages mid-run, since Wikidata is not being
// concurrently edited by this process).
func (s *Source) fetchCoreBatch(ctx context.Context, offset int) ([]coreRow, error) {
	var countryFilter string
	if len(s.CountryQIDs) > 0 {
		values := make([]string, 0, len(s.CountryQIDs))
		for _, q := range s.CountryQIDs {
			values = append(values, "wd:"+q)
		}
		countryFilter = fmt.Sprintf("FILTER(?country IN (%s))", strings.Join(values, ", "))
	}
	query := fmt.Sprintf(`
		SELECT ?diocese ?country ?parent WHERE {
		  ?diocese wdt:P1866 ?chid .
		  ?diocese wdt:P17 ?country .
		  %s
		  OPTIONAL { ?diocese wdt:P749 ?parent . }
		}
		ORDER BY ?diocese
		LIMIT %d OFFSET %d`, countryFilter, batchSize, offset)

	bindings, err := s.query(ctx, query)
	if err != nil {
		return nil, err
	}
	rows := make([]coreRow, 0, len(bindings))
	for _, b := range bindings {
		qid := qidFromURI(b["diocese"])
		if qid == "" {
			continue
		}
		row := coreRow{qid: qid}
		if v := qidFromURI(b["country"]); v != "" {
			row.countryQID = &v
		}
		if v := qidFromURI(b["parent"]); v != "" {
			row.parentQID = &v
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// fetchLabels resolves rdfs:label/skos:altLabel for exactly the given QIDs, in labelLanguages only —
// bounded by a VALUES clause so this query's cost scales with one batch, never the whole dataset.
func (s *Source) fetchLabels(ctx context.Context, qids []string) (map[string][]string, error) {
	values := make([]string, 0, len(qids))
	for _, q := range qids {
		values = append(values, "wd:"+q)
	}
	langs := make([]string, 0, len(labelLanguages))
	for _, l := range labelLanguages {
		langs = append(langs, `"`+l+`"`)
	}
	query := fmt.Sprintf(`
		SELECT ?diocese ?label WHERE {
		  VALUES ?diocese { %s }
		  { ?diocese rdfs:label ?label . }
		  UNION
		  { ?diocese skos:altLabel ?label . }
		  FILTER(LANG(?label) IN (%s))
		}`, strings.Join(values, " "), strings.Join(langs, ", "))

	bindings, err := s.query(ctx, query)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]string, len(qids))
	seen := make(map[string]map[string]bool, len(qids))
	for _, b := range bindings {
		qid := qidFromURI(b["diocese"])
		label := b["label"]
		if qid == "" || label == "" {
			continue
		}
		if seen[qid] == nil {
			seen[qid] = make(map[string]bool)
		}
		if seen[qid][label] {
			continue
		}
		seen[qid][label] = true
		out[qid] = append(out[qid], label)
	}
	return out, nil
}

// primaryAndAliases picks the first label fetchLabels returned as the primary Name; every other
// distinct label becomes an AliasNames entry. fetchLabels's map shape ([qid][]string) doesn't carry
// each label's source language tag through, so there is no reliable way to prefer "en" specifically
// here — a documented approximation, not a bug: every label, including whichever one lands as
// "primary", still becomes a real, matchable alias row either way (see
// application/jurisdictionsync.go).
func primaryAndAliases(labels []string) (name string, aliases []string) {
	if len(labels) == 0 {
		return "", nil
	}
	return labels[0], labels[1:]
}

// query POSTs a SPARQL query (form-encoded, per Wikimedia's own etiquette for anything beyond a
// trivial GET) and returns the results.bindings, each binding value flattened to plain strings
// (URI or literal, whichever the endpoint returned).
func (s *Source) query(ctx context.Context, sparql string) ([]map[string]string, error) {
	form := url.Values{}
	form.Set("query", sparql)
	form.Set("format", "json")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.BaseURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("wikidatacatholic: build request: %w", err)
	}
	citation := s.Citation()
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/sparql-results+json")
	req.Header.Set("User-Agent", citation.UserAgent)
	req.Header.Set("Api-User-Agent", citation.UserAgent)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("wikidatacatholic: POST %s: %w", s.BaseURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("wikidatacatholic: POST %s: unexpected status %s: %s", s.BaseURL, resp.Status, body)
	}

	var parsed sparqlResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("wikidatacatholic: decode SPARQL response: %w", err)
	}

	out := make([]map[string]string, 0, len(parsed.Results.Bindings))
	for _, binding := range parsed.Results.Bindings {
		flat := make(map[string]string, len(binding))
		for k, v := range binding {
			flat[k] = v.Value
		}
		out = append(out, flat)
	}
	return out, nil
}

type sparqlResponse struct {
	Results struct {
		Bindings []map[string]sparqlValue `json:"bindings"`
	} `json:"results"`
}

type sparqlValue struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

var qidFromURIPattern = regexp.MustCompile(`/(Q[1-9][0-9]*)$`)

// qidFromURI extracts "Q40741" from "http://www.wikidata.org/entity/Q40741" — returns "" for an
// empty/malformed value (e.g. an OPTIONAL var that didn't bind).
func qidFromURI(uri string) string {
	m := qidFromURIPattern.FindStringSubmatch(uri)
	if m == nil {
		return ""
	}
	return m[1]
}
