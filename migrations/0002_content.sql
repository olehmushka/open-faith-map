-- 0002_content — M3 (docs/modules/content.md). Sites, documents (pages only in M3 — post/event are
-- schema-ready, app-rejected until M4), blocks, and the block-type catalog, seeded with the MVP set.
-- congregation_unit_rid is an opaque go-oikumenea Unit RID (TEXT, no cross-schema FK —
-- conventions.md). U6 resolved: plain uuid PKs, matching registration_requests' precedent, not the
-- composed-URN RID scheme (unshipped in go-oikumenea's actually-deployed migrations — see
-- conventions.md's corrected RID entry).
--
-- kind/state/status values match api/content.conjure.yml's enum values exactly (uppercase), the
-- same convention registration_requests.status already uses — no case conversion needed anywhere
-- across the transport/domain/adapters boundary.
--
-- Design call: "columns" is the one MVP block type with real nesting, but content_blocks has no
-- parent_block_id — its children live as inline JSON *inside* the block's own data, validated as
-- part of the same json_schema, never as additional content_blocks rows (docs/modules/content.md).

CREATE TABLE openfaithmap.content_sites (
  id                     uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  congregation_unit_rid  text NOT NULL,
  slug                   text NOT NULL,
  theme                  jsonb NOT NULL DEFAULT '{}',
  created_at             timestamptz NOT NULL DEFAULT now(),
  updated_at             timestamptz NOT NULL DEFAULT now(),
  deleted_at             timestamptz
);

CREATE UNIQUE INDEX content_sites_unit_idx ON openfaithmap.content_sites (congregation_unit_rid) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX content_sites_slug_idx ON openfaithmap.content_sites (slug) WHERE deleted_at IS NULL;

CREATE TRIGGER content_sites_set_updated_at
  BEFORE UPDATE ON openfaithmap.content_sites
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();

CREATE TABLE openfaithmap.content_block_types (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  code        text NOT NULL,
  name        text NOT NULL,
  json_schema jsonb NOT NULL,
  status      text NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'RETIRED')),
  sort_order  integer NOT NULL DEFAULT 0,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now(),
  deleted_at  timestamptz
);

CREATE UNIQUE INDEX content_block_types_code_idx ON openfaithmap.content_block_types (code) WHERE deleted_at IS NULL;

CREATE TRIGGER content_block_types_set_updated_at
  BEFORE UPDATE ON openfaithmap.content_block_types
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();

CREATE TABLE openfaithmap.content_documents (
  id                      uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  site_id                 uuid NOT NULL REFERENCES openfaithmap.content_sites (id),
  kind                    text NOT NULL CHECK (kind IN ('PAGE', 'POST', 'EVENT')),
  translation_group_id    uuid NOT NULL,
  locale                  text NOT NULL,
  parent_document_id      uuid REFERENCES openfaithmap.content_documents (id),
  slug                    text NOT NULL,
  state                   text NOT NULL DEFAULT 'DRAFT' CHECK (state IN ('DRAFT', 'PUBLISHED', 'UNLISTED')),
  published_at            timestamptz,
  event_starts_at         timestamptz,
  event_ends_at           timestamptz,
  event_recurrence_rrule  text,
  created_at              timestamptz NOT NULL DEFAULT now(),
  updated_at              timestamptz NOT NULL DEFAULT now(),
  deleted_at              timestamptz,

  CONSTRAINT content_documents_parent_pages_only CHECK (parent_document_id IS NULL OR kind = 'PAGE')
);

CREATE UNIQUE INDEX content_documents_slug_idx
  ON openfaithmap.content_documents (site_id, kind, locale, slug) WHERE deleted_at IS NULL;
CREATE INDEX content_documents_translation_group_idx ON openfaithmap.content_documents (translation_group_id);
CREATE INDEX content_documents_parent_idx ON openfaithmap.content_documents (parent_document_id);
CREATE INDEX content_documents_site_state_idx ON openfaithmap.content_documents (site_id, kind, state);

CREATE TRIGGER content_documents_set_updated_at
  BEFORE UPDATE ON openfaithmap.content_documents
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();

CREATE TABLE openfaithmap.content_blocks (
  id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  document_id    uuid NOT NULL REFERENCES openfaithmap.content_documents (id) ON DELETE CASCADE,
  block_type_id  uuid NOT NULL REFERENCES openfaithmap.content_block_types (id),
  position       integer NOT NULL,
  data           jsonb NOT NULL,
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now(),
  deleted_at     timestamptz
);

CREATE UNIQUE INDEX content_blocks_position_idx ON openfaithmap.content_blocks (document_id, position) WHERE deleted_at IS NULL;

CREATE TRIGGER content_blocks_set_updated_at
  BEFORE UPDATE ON openfaithmap.content_blocks
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();

-- MVP block-type catalog seed (13 rows, content.md's own list, sort_order = authoring order).
INSERT INTO openfaithmap.content_block_types (code, name, sort_order, json_schema) VALUES
('heading', 'Heading', 10, '{"type":"object","required":["level","text"],"additionalProperties":false,"properties":{
  "level":{"type":"integer","minimum":1,"maximum":6},"text":{"type":"string","minLength":1}}}'),
('paragraph', 'Paragraph', 20, '{"type":"object","required":["text"],"additionalProperties":false,"properties":{
  "text":{"type":"string","minLength":1}}}'),
('image', 'Image', 30, '{"type":"object","required":["url"],"additionalProperties":false,"properties":{
  "url":{"type":"string","format":"uri"},"alt":{"type":"string"},"caption":{"type":"string"}}}'),
('gallery', 'Gallery', 40, '{"type":"object","required":["images"],"additionalProperties":false,"properties":{
  "images":{"type":"array","minItems":1,"items":{"type":"object","required":["url"],
    "properties":{"url":{"type":"string","format":"uri"},"alt":{"type":"string"}}}}}}'),
('youtube_embed', 'YouTube embed', 50, '{"type":"object","required":["videoId"],"additionalProperties":false,"properties":{
  "videoId":{"type":"string","minLength":1},"title":{"type":"string"}}}'),
('social_embed', 'Social embed', 60, '{"type":"object","required":["platform","url"],"additionalProperties":false,"properties":{
  "platform":{"type":"string","enum":["facebook","instagram","twitter","tiktok"]},
  "url":{"type":"string","format":"uri"}}}'),
('button', 'Button', 70, '{"type":"object","required":["label","href"],"additionalProperties":false,"properties":{
  "label":{"type":"string","minLength":1},"href":{"type":"string"},
  "style":{"type":"string","enum":["primary","secondary"]}}}'),
('contact_info', 'Contact info', 80, '{"type":"object","additionalProperties":false,"properties":{
  "address":{"type":"string"},"phone":{"type":"string"},"email":{"type":"string","format":"email"},
  "hours":{"type":"string"}}}'),
('map_embed', 'Map embed', 90, '{"type":"object","required":["latitude","longitude"],"additionalProperties":false,"properties":{
  "latitude":{"type":"number","minimum":-90,"maximum":90},
  "longitude":{"type":"number","minimum":-180,"maximum":180},"zoom":{"type":"integer","minimum":1,"maximum":20}}}'),
('divider', 'Divider', 100, '{"type":"object","additionalProperties":false,"properties":{
  "style":{"type":"string","enum":["line","space"]}}}'),
('staff_card', 'Staff card', 110, '{"type":"object","required":["name"],"additionalProperties":false,"properties":{
  "name":{"type":"string","minLength":1},"title":{"type":"string"},
  "photoUrl":{"type":"string","format":"uri"},"bio":{"type":"string"}}}'),
('quote', 'Quote', 120, '{"type":"object","required":["text"],"additionalProperties":false,"properties":{
  "text":{"type":"string","minLength":1},"attribution":{"type":"string"}}}'),
('columns', 'Columns', 130, '{"type":"object","required":["columns"],"additionalProperties":false,"properties":{
  "columns":{"type":"array","minItems":2,"maxItems":4,"items":{"type":"object","required":["blocks"],
    "properties":{"widthFraction":{"type":"number","minimum":0,"maximum":1},
      "blocks":{"type":"array","items":{"type":"object","required":["blockTypeCode","data"],
        "properties":{"blockTypeCode":{"type":"string"},"data":{"type":"object"}}}}}}}}}');
