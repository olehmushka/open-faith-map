// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

// M14.5: curation metadata for the block inserter — grouping and a one-line description per block
// type. Deliberately frontend-only (no `category`/`description` column on content_block_types):
// this milestone's own stage board marks Backend/Migrated as not-applicable, and a real schema
// column is M14.13's job (runtime catalog admin) once block types stop being migration-seeded.
// A code missing from this map (e.g. a future type added at runtime before M14.13 ships) degrades
// to "other" with no description — the inserter still lists it by name, never hides or crashes.

export type BlockCategory = "text" | "media" | "layout" | "info" | "other";

export const BLOCK_CATEGORY_ORDER: readonly BlockCategory[] = ["text", "media", "layout", "info", "other"];

export const BLOCK_CATEGORY_LABELS: Record<BlockCategory, string> = {
  text: "Text",
  media: "Media",
  layout: "Layout",
  info: "Info & contact",
  other: "Other",
};

interface BlockCatalogEntry {
  category: BlockCategory;
  description: string;
}

const BLOCK_CATALOG: Record<string, BlockCatalogEntry> = {
  heading: { category: "text", description: "A section title, in a few sizes." },
  paragraph: { category: "text", description: "A block of body text." },
  quote: { category: "text", description: "A pull quote or testimonial, with optional attribution." },
  list: { category: "text", description: "A bulleted or numbered list." },
  image: { category: "media", description: "A single photo with alt text." },
  gallery: { category: "media", description: "A grid of several photos." },
  youtube_embed: { category: "media", description: "An embedded YouTube video." },
  social_embed: { category: "media", description: "An embedded post from a supported social platform." },
  button: { category: "layout", description: "A call-to-action link styled as a button." },
  columns: { category: "layout", description: "Two or more side-by-side columns, each holding its own blocks." },
  divider: { category: "layout", description: "A horizontal rule separating sections." },
  contact_info: { category: "info", description: "Address, phone, and email, laid out for quick scanning." },
  map_embed: { category: "info", description: "An embedded map centered on an address." },
  staff_card: { category: "info", description: "A staff or clergy member's photo, name, and bio." },
};

export function blockCatalogEntry(code: string): BlockCatalogEntry {
  return BLOCK_CATALOG[code] ?? { category: "other", description: "" };
}
