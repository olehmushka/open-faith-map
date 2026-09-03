-- 0031_content_form_submissions — M14.16 (docs/architecture/decisions.md#d-inappinbox). A genuinely
-- anonymous contact-form write lands here, read back through openfaithmap-admin's Messages screen.
-- No email is ever sent (D-InviteLinkMVP's own precedent).
--
-- No soft-delete, no updated_at/set_updated_at trigger: mirrors content_site_nav_items'
-- (migrations/0027) own precedent for a table whose rows are never mutated in place — a submission
-- is written once by ContentPublicService.submitContactForm and only ever read afterward, never
-- updated. name/email are nullable (a visitor may leave either blank); message is required — the
-- one field genuinely useless empty.
--
-- Also seeds a new contact_form block type into the existing catalog (migrations/0002_content.sql),
-- sorted after columns (130): the block only configures the surrounding copy (an optional heading/
-- description) — the visitor-entered name/email/message never lives in content_blocks.data, only
-- in this new table, written through the public submitContactForm endpoint.

CREATE TABLE openfaithmap.content_form_submissions (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  site_id     uuid NOT NULL REFERENCES openfaithmap.content_sites (id) ON DELETE CASCADE,
  name        text,
  email       text,
  message     text NOT NULL,
  created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX content_form_submissions_site_created_idx
  ON openfaithmap.content_form_submissions (site_id, created_at DESC);

INSERT INTO openfaithmap.content_block_types (code, name, sort_order, json_schema, ui_schema) VALUES
('contact_form', 'Contact form', 140,
 '{"type":"object","additionalProperties":false,"properties":{
  "heading":{"type":"string"},"description":{"type":"string"}}}',
 '{"fields":[
  {"name":"heading","widget":"text","label":"Heading"},
  {"name":"description","widget":"textarea","label":"Description"}
]}');
