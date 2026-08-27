// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"encoding/json"
	"net/url"
	"regexp"
)

// googleDriveFileIDRe matches a Drive share link's /file/d/<id>/... path.
var googleDriveFileIDRe = regexp.MustCompile(`^/file/d/([a-zA-Z0-9_-]+)`)

// normalizeMediaURL rewrites a known share-link host (Google Drive, Dropbox, OneDrive) to its
// direct-content form, per D-ExternalMediaOnly/DS-OFM-17 (M14.3). Pure string rewriting only — no
// network call is ever made against an admin-supplied URL, which is why OneDrive's short
// "1drv.ms" links (resolvable only by following a redirect) are left unchanged rather than
// resolved: doing that would be exactly the SSRF surface this arc otherwise avoids entirely. Any
// host this function doesn't recognize also passes through unchanged.
func normalizeMediaURL(raw string) (normalized string, changed bool) {
	u, err := url.Parse(raw)
	if err != nil {
		return raw, false
	}
	switch u.Hostname() {
	case "drive.google.com":
		if m := googleDriveFileIDRe.FindStringSubmatch(u.Path); m != nil {
			return "https://drive.google.com/uc?export=view&id=" + m[1], true
		}
		if u.Path == "/open" {
			if id := u.Query().Get("id"); id != "" {
				return "https://drive.google.com/uc?export=view&id=" + id, true
			}
		}
	case "dropbox.com", "www.dropbox.com":
		q := u.Query()
		if q.Get("raw") == "1" {
			return raw, false
		}
		q.Del("dl")
		q.Set("raw", "1")
		u.Scheme = "https"
		u.Host = "www.dropbox.com"
		u.RawQuery = q.Encode()
		return u.String(), true
	case "onedrive.live.com":
		if u.Path == "/redir" {
			u.Path = "/download"
			return u.String(), true
		}
	}
	return raw, false
}

// normalizeBlockMediaURLs rewrites image.url / gallery.images[].url in place on a block's raw
// JSON data for known share-link hosts, storing the pre-rewrite value in a new "originalUrl"
// field only when a rewrite actually happened — per D-ExternalMediaOnly's consequence that the
// original URL is preserved alongside the normalized one, so a future normalizer fix is a
// re-derivation, not a data-loss event.
//
// Only top-level image/gallery blocks are rewritten. Nested blocks under a "columns" block's
// data.columns[].blocks[] already bypass content_block_types.json_schema entirely
// (blockvalidation.go's own "columns" case), so they are left as-is here too — the same
// deliberately-accepted gap M14.2's migration documented for richText.
//
// Malformed data is passed through unchanged; the structural json_schema pass in
// validateBlockData reports that.
func normalizeBlockMediaURLs(blockTypeCode string, data []byte) ([]byte, error) {
	switch blockTypeCode {
	case "image":
		var obj map[string]any
		if err := json.Unmarshal(data, &obj); err != nil {
			return data, nil
		}
		raw, ok := obj["url"].(string)
		if !ok || raw == "" {
			return data, nil
		}
		norm, changed := normalizeMediaURL(raw)
		if !changed {
			return data, nil
		}
		obj["originalUrl"] = raw
		obj["url"] = norm
		return json.Marshal(obj)
	case "gallery":
		var obj map[string]any
		if err := json.Unmarshal(data, &obj); err != nil {
			return data, nil
		}
		images, _ := obj["images"].([]any)
		changedAny := false
		for _, rawImg := range images {
			img, ok := rawImg.(map[string]any)
			if !ok {
				continue
			}
			raw, ok := img["url"].(string)
			if !ok || raw == "" {
				continue
			}
			norm, changed := normalizeMediaURL(raw)
			if !changed {
				continue
			}
			img["originalUrl"] = raw
			img["url"] = norm
			changedAny = true
		}
		if !changedAny {
			return data, nil
		}
		return json.Marshal(obj)
	default:
		return data, nil
	}
}
