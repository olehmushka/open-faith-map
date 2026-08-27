-- 0024_content_ui_schema — M14.4. New content_block_types.ui_schema JSONB: widget hints, labels,
-- help text, and field order, sitting beside the existing json_schema (WordPress's block.json
-- lesson — a block's data shape and its editing affordances belong in one declaration). A generic
-- form renderer in web/apps/admin (block-data-form.tsx) builds each block's editor from the pair,
-- so a block type added at runtime (a future M14.13 catalog endpoint) renders a usable form with
-- no admin-app code change.
--
-- Shape: {"fields":[{"name","widget","label","help"?,"options"?,"min"?,"max"?,"step"?,"minItems"?,
-- "maxItems"?,"itemLabel"?,"itemFields"?}]}. widget is one of
-- text|url|number|select|textarea|array|block-list.
--
-- "textarea" is also used for every richText field (heading.text, paragraph.text, quote.text,
-- staff_card.bio, list.content) — M14.4 keeps richText a schema-aware JSON textarea rather than
-- building a WYSIWYG editor. Named, documented gap, decided with the owner, not silently dropped
-- (no rich-text-editor library exists anywhere in this repo yet; a real editor is a future
-- milestone, not yet scheduled).
--
-- "block-list" is the one recursive widget: the renderer looks each item's own blockTypeCode up in
-- the full block-type catalog and re-invokes itself with THAT type's own json_schema+ui_schema —
-- the only way "columns" nesting can render as a form at all, since data.columns[].blocks[].data
-- has never been schema-constrained beyond {"type":"object"} (see 0002_content.sql's own
-- design-call comment). A schema-shape validation failure inside a nested block still can only be
-- attributed to the outer "columns" field as a whole (blockvalidation.go's validateBlockData never
-- descends into nested block data) — the form highlights the whole Columns field group in that
-- case, not the specific nested field. validateBlockURLs's separate, existing recursive
-- URL-allowlist pass is unchanged by this milestone.
--
-- NOT NULL DEFAULT '{}' (matching content_sites.theme's own precedent one table over, this same
-- migration file's ancestor 0002_content.sql) rather than nullable: any block type inserted later
-- without an explicit ui_schema — including one added before the admin UI grows its own ui_schema
-- editor — still gets a renderable (degenerate, but never null-crashing) fallback.
--
-- Pure column-add plus one UPDATE per existing row — no content_blocks data migration needed,
-- since ui_schema only describes how to edit data, never validates or reshapes it.

ALTER TABLE openfaithmap.content_block_types ADD COLUMN ui_schema jsonb NOT NULL DEFAULT '{}'::jsonb;

UPDATE openfaithmap.content_block_types SET ui_schema = '{"fields":[
  {"name":"level","widget":"number","label":"Heading level","help":"1 (largest) through 6 (smallest).","min":1,"max":6},
  {"name":"text","widget":"textarea","label":"Text","help":"Rich-text node JSON — a visual editor is a future milestone."}
]}' WHERE code = 'heading';

UPDATE openfaithmap.content_block_types SET ui_schema = '{"fields":[
  {"name":"text","widget":"textarea","label":"Text","help":"Rich-text node JSON — a visual editor is a future milestone."}
]}' WHERE code = 'paragraph';

UPDATE openfaithmap.content_block_types SET ui_schema = '{"fields":[
  {"name":"url","widget":"url","label":"Image URL"},
  {"name":"alt","widget":"text","label":"Alt text","help":"Required — describes the image for screen readers and search."},
  {"name":"caption","widget":"text","label":"Caption"}
]}' WHERE code = 'image';

UPDATE openfaithmap.content_block_types SET ui_schema = '{"fields":[
  {"name":"images","widget":"array","label":"Images","help":"At least one image.","minItems":1,"itemLabel":"Image","itemFields":[
    {"name":"url","widget":"url","label":"Image URL"},
    {"name":"alt","widget":"text","label":"Alt text","help":"Required."}
  ]}
]}' WHERE code = 'gallery';

UPDATE openfaithmap.content_block_types SET ui_schema = '{"fields":[
  {"name":"videoId","widget":"text","label":"YouTube video ID","help":"The v= value from the video''s URL, not the whole URL."},
  {"name":"title","widget":"text","label":"Title"}
]}' WHERE code = 'youtube_embed';

UPDATE openfaithmap.content_block_types SET ui_schema = '{"fields":[
  {"name":"platform","widget":"select","label":"Platform","options":[
    {"value":"facebook","label":"Facebook"},{"value":"instagram","label":"Instagram"},
    {"value":"twitter","label":"Twitter / X"},{"value":"tiktok","label":"TikTok"}
  ]},
  {"name":"url","widget":"url","label":"Post URL"}
]}' WHERE code = 'social_embed';

UPDATE openfaithmap.content_block_types SET ui_schema = '{"fields":[
  {"name":"label","widget":"text","label":"Button label"},
  {"name":"href","widget":"url","label":"Link"},
  {"name":"style","widget":"select","label":"Style","options":[
    {"value":"primary","label":"Primary"},{"value":"secondary","label":"Secondary"}
  ]}
]}' WHERE code = 'button';

UPDATE openfaithmap.content_block_types SET ui_schema = '{"fields":[
  {"name":"address","widget":"text","label":"Address"},
  {"name":"phone","widget":"text","label":"Phone"},
  {"name":"email","widget":"text","label":"Email"},
  {"name":"hours","widget":"text","label":"Hours"}
]}' WHERE code = 'contact_info';

UPDATE openfaithmap.content_block_types SET ui_schema = '{"fields":[
  {"name":"latitude","widget":"number","label":"Latitude","min":-90,"max":90,"step":0.0001},
  {"name":"longitude","widget":"number","label":"Longitude","min":-180,"max":180,"step":0.0001},
  {"name":"zoom","widget":"number","label":"Zoom","min":1,"max":20}
]}' WHERE code = 'map_embed';

UPDATE openfaithmap.content_block_types SET ui_schema = '{"fields":[
  {"name":"style","widget":"select","label":"Style","options":[
    {"value":"line","label":"Line"},{"value":"space","label":"Blank space"}
  ]}
]}' WHERE code = 'divider';

UPDATE openfaithmap.content_block_types SET ui_schema = '{"fields":[
  {"name":"name","widget":"text","label":"Name"},
  {"name":"title","widget":"text","label":"Title"},
  {"name":"photoUrl","widget":"url","label":"Photo URL"},
  {"name":"bio","widget":"textarea","label":"Bio","help":"Rich-text node JSON — a visual editor is a future milestone."}
]}' WHERE code = 'staff_card';

UPDATE openfaithmap.content_block_types SET ui_schema = '{"fields":[
  {"name":"text","widget":"textarea","label":"Quote text","help":"Rich-text node JSON — a visual editor is a future milestone."},
  {"name":"attribution","widget":"text","label":"Attribution"}
]}' WHERE code = 'quote';

UPDATE openfaithmap.content_block_types SET ui_schema = '{"fields":[
  {"name":"columns","widget":"array","label":"Columns","minItems":2,"maxItems":4,"itemLabel":"Column","itemFields":[
    {"name":"widthFraction","widget":"number","label":"Width fraction","help":"0–1, optional.","min":0,"max":1,"step":0.05},
    {"name":"blocks","widget":"block-list","label":"Blocks in this column"}
  ]}
]}' WHERE code = 'columns';

UPDATE openfaithmap.content_block_types SET ui_schema = '{"fields":[
  {"name":"content","widget":"textarea","label":"Content","help":"Rich-text node JSON — a visual editor is a future milestone."}
]}' WHERE code = 'list';
