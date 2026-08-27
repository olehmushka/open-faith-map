// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/olehmushka/open-faith-map/internal/content/domain"
	"github.com/santhosh-tekuri/jsonschema/v6"
	jsonschemakind "github.com/santhosh-tekuri/jsonschema/v6/kind"
)

// validateBlockData resolves blockTypeCode to an active block-type row and validates data against
// its json_schema — content.md's "blocks always schema-valid at write time" invariant. A retired or
// unknown code is domain.ErrBlockTypeNotFound; a schema violation is
// domain.BlockDataInvalidError{BlockTypeCode, Position, Field} (never the raw validator message,
// which could echo arbitrary submitted content into a safe-arg — see transport/errors.go; Field is
// filtered through topLevelFieldFromValidationError for the same reason). Once structurally valid,
// D-PublicSiteCSP's URL-scheme/embed-host allowlist runs as a second pass (validateBlockURLs) —
// json_schema's own "format":"uri" keyword accepts a "javascript:" URI just fine, so it cannot be
// the enforcement point for this.
func validateBlockData(blockType domain.BlockType, position int, data []byte) error {
	schemaDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(blockType.JSONSchema))
	if err != nil {
		return fmt.Errorf("block type %q: parse json_schema: %w", blockType.Code, err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(blockType.Code, schemaDoc); err != nil {
		return fmt.Errorf("block type %q: add schema resource: %w", blockType.Code, err)
	}
	sch, err := compiler.Compile(blockType.Code)
	if err != nil {
		return fmt.Errorf("block type %q: compile schema: %w", blockType.Code, err)
	}

	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return &domain.BlockDataInvalidError{BlockTypeCode: blockType.Code, Position: position}
	}
	if err := sch.Validate(instance); err != nil {
		return &domain.BlockDataInvalidError{
			BlockTypeCode: blockType.Code,
			Position:      position,
			Field:         topLevelFieldFromValidationError(schemaDoc, err),
		}
	}
	if err := validateBlockURLs(blockType, position, instance); err != nil {
		return err
	}
	return nil
}

// topLevelFieldFromValidationError walks a jsonschema/v6 validation error tree (a nest of Causes,
// since a top-level "oneOf"/"required" failure groups its sub-failures) for the first
// InstanceLocation whose first path segment names one of the block type's own declared top-level
// "properties" keys — a small, fixed, developer-authored set known from schemaDoc itself, never
// from untrusted instance data. This is the one place an attacker-influenced string (e.g. an
// unexpected property name, on a schema with additionalProperties:false) could otherwise reach a
// Conjure safe-arg, so any path segment not in that set is dropped rather than surfaced. Deeper
// paths (array indices, nested object keys) are intentionally not resolved this way — a "columns"
// block's nested failures land on the bare "columns" field, and array-item failures land on the
// bare array field name (e.g. "images"); BlockUrlNotAllowedError already reports more precise
// per-item field names for the separate URL-allowlist pass.
func topLevelFieldFromValidationError(schemaDoc any, err error) string {
	props := topLevelPropertyNames(schemaDoc)

	var walk func(*jsonschema.ValidationError) string
	walk = func(ve *jsonschema.ValidationError) string {
		// A "required" failure's own InstanceLocation is the *parent* object (root, for a
		// top-level required field like image.alt) — the missing property name lives in
		// ErrorKind.Missing instead, so it has to be appended before taking the first segment.
		loc := ve.InstanceLocation
		if req, ok := ve.ErrorKind.(*jsonschemakind.Required); ok && len(req.Missing) > 0 {
			loc = append(append([]string{}, loc...), req.Missing[0])
		}
		if len(loc) > 0 {
			if seg := loc[0]; props[seg] {
				return seg
			}
		}
		for _, cause := range ve.Causes {
			if f := walk(cause); f != "" {
				return f
			}
		}
		return ""
	}

	var ve *jsonschema.ValidationError
	if errors.As(err, &ve) {
		return walk(ve)
	}
	return ""
}

func topLevelPropertyNames(schemaDoc any) map[string]bool {
	out := map[string]bool{}
	obj, _ := schemaDoc.(map[string]any)
	props, _ := obj["properties"].(map[string]any)
	for k := range props {
		out[k] = true
	}
	return out
}

// allowedURLSchemes is D-PublicSiteCSP's scheme allowlist, applied to every URL-bearing block
// field repo-wide. Deliberately small: no "ftp", no "data", and no "javascript" — javascript: is
// exactly the live stored-XSS this closes.
var allowedURLSchemes = map[string]bool{
	"https": true, "http": true, "mailto": true, "tel": true,
}

// socialEmbedHosts is the embed-host allowlist for social_embed.url, keyed by its declared
// platform (migrations/0002_content.sql's own enum: facebook/instagram/twitter/tiktok).
var socialEmbedHosts = map[string][]string{
	"facebook":  {"www.facebook.com", "facebook.com", "fb.watch"},
	"instagram": {"www.instagram.com", "instagram.com"},
	"twitter":   {"twitter.com", "x.com"},
	"tiktok":    {"www.tiktok.com", "tiktok.com"},
}

// validateBlockURLs walks every URL-bearing field for blockType.Code and rejects a disallowed
// scheme/host with a typed, field-naming error — belt-and-braces with the render-time re-check in
// web/apps/web/lib/block-security.ts, which exists because rows written before this landed, and
// any future block type reintroducing an unguarded field, both bypass this write-time gate.
//
// youtube_embed.videoId is not itself a URL (the embed src is server-constructed at render time),
// contact_info.email is format:"email" not a URL-scheme concern, and map_embed's link is built
// server-side from numeric lat/lng — none of the three have an admin-supplied URL to check here.
func validateBlockURLs(blockType domain.BlockType, position int, instance any) error {
	data, _ := instance.(map[string]any)
	if data == nil {
		return nil // structural validation already caught non-object data
	}
	fail := func(field string) error {
		return &domain.BlockUrlNotAllowedError{BlockTypeCode: blockType.Code, Position: position, Field: field}
	}
	checkScheme := func(field string, v any) error {
		s, _ := v.(string)
		if s == "" {
			return nil // schema's own required/minLength already covers absence
		}
		u, err := url.Parse(s)
		if err != nil || u.Scheme == "" || !allowedURLSchemes[strings.ToLower(u.Scheme)] {
			return fail(field)
		}
		return nil
	}

	// checkRichTextLinks walks a richText node array (D-RichTextNodes) and runs every "link" mark's
	// href through checkScheme, recursing into "list" nodes' items — a link mark is the one URL
	// value nested inside a tree rather than living directly on the block, so it needs its own walk
	// instead of a single field lookup. Reports the top-level field name (e.g. "text"), matching the
	// existing "as precise as it's cheap to be" precedent (images[%d].url) rather than a full node
	// path.
	var checkRichTextLinks func(field string, value any) error
	checkRichTextLinks = func(field string, value any) error {
		nodes, _ := value.([]any)
		for _, rawNode := range nodes {
			node, _ := rawNode.(map[string]any)
			if node == nil {
				continue
			}
			switch node["type"] {
			case "text":
				marks, _ := node["marks"].([]any)
				for _, rawMark := range marks {
					mark, _ := rawMark.(map[string]any)
					if mark == nil || mark["type"] != "link" {
						continue
					}
					if err := checkScheme(field, mark["href"]); err != nil {
						return err
					}
				}
			case "list":
				items, _ := node["items"].([]any)
				for _, rawItem := range items {
					item, _ := rawItem.(map[string]any)
					if item == nil {
						continue
					}
					if err := checkRichTextLinks(field, item["content"]); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}

	switch blockType.Code {
	case "image":
		if err := checkScheme("url", data["url"]); err != nil {
			return err
		}
	case "gallery":
		images, _ := data["images"].([]any)
		for i, raw := range images {
			img, _ := raw.(map[string]any)
			if err := checkScheme(fmt.Sprintf("images[%d].url", i), img["url"]); err != nil {
				return err
			}
		}
	case "button":
		if err := checkScheme("href", data["href"]); err != nil {
			return err
		}
	case "paragraph", "heading", "quote":
		if err := checkRichTextLinks("text", data["text"]); err != nil {
			return err
		}
	case "list":
		if err := checkRichTextLinks("content", data["content"]); err != nil {
			return err
		}
	case "staff_card":
		if err := checkScheme("photoUrl", data["photoUrl"]); err != nil {
			return err
		}
		if err := checkRichTextLinks("bio", data["bio"]); err != nil {
			return err
		}
	case "social_embed":
		if err := checkScheme("url", data["url"]); err != nil {
			return err
		}
		rawURL, _ := data["url"].(string)
		if rawURL != "" {
			platform, _ := data["platform"].(string)
			u, err := url.Parse(rawURL)
			if err != nil || !hostAllowed(u.Host, socialEmbedHosts[platform]) {
				return fail("url")
			}
		}
	case "columns":
		// Nested blocks are declared as "data":{"type":"object"} with no further schema
		// constraint (migrations/0002_content.sql) — they bypass the outer json_schema pass
		// entirely, so a button/image nested under columns must be recursed into explicitly or
		// it goes unchecked by both this pass and the structural one.
		columns, _ := data["columns"].([]any)
		for _, rawCol := range columns {
			col, _ := rawCol.(map[string]any)
			blocks, _ := col["blocks"].([]any)
			for _, rawNB := range blocks {
				nb, _ := rawNB.(map[string]any)
				nbCode, _ := nb["blockTypeCode"].(string)
				nbData, _ := nb["data"].(map[string]any)
				if err := validateBlockURLs(domain.BlockType{Code: nbCode}, position, nbData); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// hostAllowed is a case-insensitive exact match against an explicit host list — no wildcard/suffix
// matching, per D-PublicSiteCSP's own "named" host-set wording.
func hostAllowed(host string, allowed []string) bool {
	host = strings.ToLower(host)
	for _, h := range allowed {
		if host == strings.ToLower(h) {
			return true
		}
	}
	return false
}
