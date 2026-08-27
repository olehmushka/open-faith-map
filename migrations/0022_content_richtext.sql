-- 0022_content_richtext — M14.2. Replaces the plain-string text/bio fields on paragraph, heading,
-- quote and staff_card with a shared `richText` node array (D-RichTextNodes,
-- docs/architecture/decisions.md): an ordered array of `text` runs carrying bold/italic/link marks,
-- plus `list`/`listItem` nodes (a list item's own content is itself a richText array, so nesting
-- works structurally for free). There is no HTML string anywhere in this pipeline, so there is no
-- HTML parser and no sanitizer to get wrong — the renderer (web/apps/web/lib/rich-text.tsx) maps
-- node types straight to React elements. A `link` mark's href goes through the exact same
-- URL-scheme allowlist as every other URL field (internal/content/application/blockvalidation.go's
-- validateBlockURLs), not a separate path.
--
-- content_block_types.json_schema has no cross-row $ref, so the richText $defs block is repeated
-- literally in each of the five schemas below rather than shared by reference.
--
-- Expand-and-data migration in one file (this repo's convention for a schema change that would
-- otherwise reject rows already in the table): each UPDATE below both loosens the json_schema *and*
-- lifts every existing plain-string text/bio value into a single-text-run node, so a schema that
-- would reject rows already in the table never actually does.
--
-- Known, deliberately-accepted gap: nested blocks inside a "columns" block's
-- data.columns[].blocks[] already bypass content_block_types.json_schema entirely (see the
-- "columns" case in blockvalidation.go / the design-call comment in 0002_content.sql) and are not
-- rewritten here. No such nested paragraph/heading/quote/staff_card fixture data exists in any
-- migration or seed today, and the product has no live congregation content yet (M14.18/deployment
-- is still blocked on U14) — so there is no real data at risk. The renderer degrades a non-array
-- legacy value to "render nothing" rather than crashing if this is ever hit.

UPDATE openfaithmap.content_block_types SET json_schema = '{"type":"object","required":["level","text"],"additionalProperties":false,"properties":{"level":{"type":"integer","minimum":1,"maximum":6},"text":{"$ref":"#/$defs/richText"}},"$defs":{"richText":{"type":"array","items":{"oneOf":[{"type":"object","required":["type","text"],"additionalProperties":false,"properties":{"type":{"enum":["text"]},"text":{"type":"string"},"marks":{"type":"array","items":{"oneOf":[{"type":"object","required":["type"],"additionalProperties":false,"properties":{"type":{"enum":["bold"]}}},{"type":"object","required":["type"],"additionalProperties":false,"properties":{"type":{"enum":["italic"]}}},{"type":"object","required":["type","href"],"additionalProperties":false,"properties":{"type":{"enum":["link"]},"href":{"type":"string"}}}]}}}},{"type":"object","required":["type","style","items"],"additionalProperties":false,"properties":{"type":{"enum":["list"]},"style":{"type":"string","enum":["bullet","ordered"]},"items":{"type":"array","items":{"type":"object","required":["type","content"],"additionalProperties":false,"properties":{"type":{"enum":["listItem"]},"content":{"$ref":"#/$defs/richText"}}}}}}]}}}}'
WHERE code = 'heading';

UPDATE openfaithmap.content_block_types SET json_schema = '{"type":"object","required":["text"],"additionalProperties":false,"properties":{"text":{"$ref":"#/$defs/richText"}},"$defs":{"richText":{"type":"array","items":{"oneOf":[{"type":"object","required":["type","text"],"additionalProperties":false,"properties":{"type":{"enum":["text"]},"text":{"type":"string"},"marks":{"type":"array","items":{"oneOf":[{"type":"object","required":["type"],"additionalProperties":false,"properties":{"type":{"enum":["bold"]}}},{"type":"object","required":["type"],"additionalProperties":false,"properties":{"type":{"enum":["italic"]}}},{"type":"object","required":["type","href"],"additionalProperties":false,"properties":{"type":{"enum":["link"]},"href":{"type":"string"}}}]}}}},{"type":"object","required":["type","style","items"],"additionalProperties":false,"properties":{"type":{"enum":["list"]},"style":{"type":"string","enum":["bullet","ordered"]},"items":{"type":"array","items":{"type":"object","required":["type","content"],"additionalProperties":false,"properties":{"type":{"enum":["listItem"]},"content":{"$ref":"#/$defs/richText"}}}}}}]}}}}'
WHERE code = 'paragraph';

UPDATE openfaithmap.content_block_types SET json_schema = '{"type":"object","required":["text"],"additionalProperties":false,"properties":{"text":{"$ref":"#/$defs/richText"},"attribution":{"type":"string"}},"$defs":{"richText":{"type":"array","items":{"oneOf":[{"type":"object","required":["type","text"],"additionalProperties":false,"properties":{"type":{"enum":["text"]},"text":{"type":"string"},"marks":{"type":"array","items":{"oneOf":[{"type":"object","required":["type"],"additionalProperties":false,"properties":{"type":{"enum":["bold"]}}},{"type":"object","required":["type"],"additionalProperties":false,"properties":{"type":{"enum":["italic"]}}},{"type":"object","required":["type","href"],"additionalProperties":false,"properties":{"type":{"enum":["link"]},"href":{"type":"string"}}}]}}}},{"type":"object","required":["type","style","items"],"additionalProperties":false,"properties":{"type":{"enum":["list"]},"style":{"type":"string","enum":["bullet","ordered"]},"items":{"type":"array","items":{"type":"object","required":["type","content"],"additionalProperties":false,"properties":{"type":{"enum":["listItem"]},"content":{"$ref":"#/$defs/richText"}}}}}}]}}}}'
WHERE code = 'quote';

UPDATE openfaithmap.content_block_types SET json_schema = '{"type":"object","required":["name"],"additionalProperties":false,"properties":{"name":{"type":"string","minLength":1},"title":{"type":"string"},"photoUrl":{"type":"string","format":"uri"},"bio":{"$ref":"#/$defs/richText"}},"$defs":{"richText":{"type":"array","items":{"oneOf":[{"type":"object","required":["type","text"],"additionalProperties":false,"properties":{"type":{"enum":["text"]},"text":{"type":"string"},"marks":{"type":"array","items":{"oneOf":[{"type":"object","required":["type"],"additionalProperties":false,"properties":{"type":{"enum":["bold"]}}},{"type":"object","required":["type"],"additionalProperties":false,"properties":{"type":{"enum":["italic"]}}},{"type":"object","required":["type","href"],"additionalProperties":false,"properties":{"type":{"enum":["link"]},"href":{"type":"string"}}}]}}}},{"type":"object","required":["type","style","items"],"additionalProperties":false,"properties":{"type":{"enum":["list"]},"style":{"type":"string","enum":["bullet","ordered"]},"items":{"type":"array","items":{"type":"object","required":["type","content"],"additionalProperties":false,"properties":{"type":{"enum":["listItem"]},"content":{"$ref":"#/$defs/richText"}}}}}}]}}}}'
WHERE code = 'staff_card';

-- New block type, appended after "columns" (sort_order 130) rather than inserted mid-sequence, so
-- no existing sort_order read has to change.
INSERT INTO openfaithmap.content_block_types (code, name, sort_order, json_schema) VALUES
('list', 'List', 140, '{"type":"object","required":["content"],"additionalProperties":false,"properties":{"content":{"$ref":"#/$defs/richText"}},"$defs":{"richText":{"type":"array","items":{"oneOf":[{"type":"object","required":["type","text"],"additionalProperties":false,"properties":{"type":{"enum":["text"]},"text":{"type":"string"},"marks":{"type":"array","items":{"oneOf":[{"type":"object","required":["type"],"additionalProperties":false,"properties":{"type":{"enum":["bold"]}}},{"type":"object","required":["type"],"additionalProperties":false,"properties":{"type":{"enum":["italic"]}}},{"type":"object","required":["type","href"],"additionalProperties":false,"properties":{"type":{"enum":["link"]},"href":{"type":"string"}}}]}}}},{"type":"object","required":["type","style","items"],"additionalProperties":false,"properties":{"type":{"enum":["list"]},"style":{"type":"string","enum":["bullet","ordered"]},"items":{"type":"array","items":{"type":"object","required":["type","content"],"additionalProperties":false,"properties":{"type":{"enum":["listItem"]},"content":{"$ref":"#/$defs/richText"}}}}}}]}}}}');

-- Data migration: lift every existing plain-string text value into a single-text-run node, for
-- paragraph/heading/quote blocks whose data.text is still a JSON string (not yet migrated).
UPDATE openfaithmap.content_blocks cb
SET data = jsonb_set(cb.data, '{text}', jsonb_build_array(jsonb_build_object('type', 'text', 'text', cb.data->'text')))
FROM openfaithmap.content_block_types bt
WHERE cb.block_type_id = bt.id
  AND bt.code IN ('paragraph', 'heading', 'quote')
  AND jsonb_typeof(cb.data->'text') = 'string';

-- Same lift for staff_card.bio, guarded on bio being present and a string since bio is optional.
UPDATE openfaithmap.content_blocks cb
SET data = jsonb_set(cb.data, '{bio}', jsonb_build_array(jsonb_build_object('type', 'text', 'text', cb.data->'bio')))
FROM openfaithmap.content_block_types bt
WHERE cb.block_type_id = bt.id
  AND bt.code = 'staff_card'
  AND jsonb_typeof(cb.data->'bio') = 'string';
