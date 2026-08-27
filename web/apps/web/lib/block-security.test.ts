// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";

import { isValidYoutubeVideoId, safeEmbedSrc, safeSocialEmbedUrl, safeUrl } from "./block-security";

describe("safeUrl", () => {
  it.each(["https://example.org", "http://example.org", "mailto:a@example.org", "tel:+15551234567"])(
    "allows %s",
    (url) => {
      expect(safeUrl(url)).toBe(url);
    },
  );

  it.each(["javascript:alert(1)", "data:text/html,<script>alert(1)</script>", "ftp://example.org/file", "vbscript:msgbox(1)"])(
    "rejects %s",
    (url) => {
      expect(safeUrl(url)).toBeUndefined();
    },
  );

  it("rejects a relative path (no scheme)", () => {
    expect(safeUrl("/some/path")).toBeUndefined();
  });

  it("rejects a malformed URL", () => {
    expect(safeUrl("not a url at all")).toBeUndefined();
  });

  it("rejects non-string input", () => {
    expect(safeUrl(undefined)).toBeUndefined();
    expect(safeUrl(null)).toBeUndefined();
    expect(safeUrl(42)).toBeUndefined();
    expect(safeUrl("")).toBeUndefined();
  });
});

describe("safeSocialEmbedUrl", () => {
  it("allows a URL on the platform's own host", () => {
    expect(safeSocialEmbedUrl("https://www.facebook.com/post/1", "facebook")).toBe("https://www.facebook.com/post/1");
    expect(safeSocialEmbedUrl("https://x.com/post/1", "twitter")).toBe("https://x.com/post/1");
  });

  it("rejects a URL on a host that doesn't match the declared platform", () => {
    expect(safeSocialEmbedUrl("https://evil.example.com/post/1", "facebook")).toBeUndefined();
  });

  it("rejects an unknown platform", () => {
    expect(safeSocialEmbedUrl("https://www.facebook.com/post/1", "myspace")).toBeUndefined();
  });

  it("rejects a disallowed scheme regardless of host", () => {
    expect(safeSocialEmbedUrl("javascript:alert(1)", "facebook")).toBeUndefined();
  });
});

describe("safeEmbedSrc", () => {
  it("allows a youtube_embed src on youtube's own host", () => {
    const src = "https://www.youtube.com/embed/dQw4w9WgXcQ";
    expect(safeEmbedSrc("youtube_embed", src)).toBe(src);
  });

  it("rejects a youtube_embed src on a different host", () => {
    expect(safeEmbedSrc("youtube_embed", "https://evil.example.com/embed/dQw4w9WgXcQ")).toBeUndefined();
  });

  it("rejects a non-https youtube_embed src", () => {
    expect(safeEmbedSrc("youtube_embed", "http://www.youtube.com/embed/dQw4w9WgXcQ")).toBeUndefined();
  });

  it("rejects an unknown block type entirely (no allowlist entry)", () => {
    expect(safeEmbedSrc("vimeo_embed", "https://player.vimeo.com/video/1")).toBeUndefined();
  });
});

describe("isValidYoutubeVideoId", () => {
  it.each(["dQw4w9WgXcQ", "abc-123_XYZ"])("accepts %s", (id) => {
    expect(isValidYoutubeVideoId(id)).toBe(true);
  });

  it.each(["../../evil", "a b", "<script>", ""])("rejects %s", (id) => {
    expect(isValidYoutubeVideoId(id)).toBe(false);
  });

  it("rejects non-string input", () => {
    expect(isValidYoutubeVideoId(undefined)).toBe(false);
  });
});
