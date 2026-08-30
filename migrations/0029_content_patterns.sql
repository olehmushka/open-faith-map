-- 0029_content_patterns — M14.13 (D-SitePatterns). content_patterns ships pre-built,
-- church-specific starting layouts with WordPress's unsynced semantics: inserting a pattern copies
-- its blocks into a document and detaches immediately — no ongoing link, no shared state, freely
-- edited afterward (docs/modules/content.md's own Entities section). Same table shape/conventions
-- as content_block_types (migrations/0002_content.sql): plain uuid PK, soft-delete, an
-- updated_at trigger.
--
-- No natural unique key like content_block_types.code exists here (a pattern is just a named
-- template, not a referenced-by-code catalog entry), so there is no unique index beyond the PK —
-- deletePattern is a soft delete via deleted_at, matching every other soft-deleted row in this
-- schema.
--
-- blocks is a full blocks snapshot, same BlockInput shape (blockTypeCode/position/data) a
-- document's own block list already uses — application.Service.CreateBlockType/CreatePattern
-- validate this shape at write time; the seed rows below are inserted directly (bypassing Go
-- validation), the same convention migrations/0002_content.sql's own block-type seed already
-- uses, built by hand against the *current* (post-M14.2 richText, post-M14.3 alt-required) schemas
-- for heading/paragraph/list/staff_card/image/button.

CREATE TABLE openfaithmap.content_patterns (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name        text NOT NULL,
  description text NOT NULL DEFAULT '',
  blocks      jsonb NOT NULL,
  sort_order  integer NOT NULL DEFAULT 0,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now(),
  deleted_at  timestamptz
);

CREATE INDEX content_patterns_sort_order_idx ON openfaithmap.content_patterns (sort_order) WHERE deleted_at IS NULL;

CREATE TRIGGER content_patterns_set_updated_at
  BEFORE UPDATE ON openfaithmap.content_patterns
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();

INSERT INTO openfaithmap.content_patterns (name, description, sort_order, blocks) VALUES
('Parish home page', 'A welcoming landing page with an intro, a photo, and a call to action.', 10,
 '[
    {"blockTypeCode":"heading","position":0,"data":{"level":1,"text":[{"type":"text","text":"Welcome to Our Parish"}]}},
    {"blockTypeCode":"paragraph","position":1,"data":{"text":[{"type":"text","text":"We are a warm, welcoming community rooted in faith and tradition. Whether you are visiting for the first time or have worshipped with us for years, we are glad you are here."}]}},
    {"blockTypeCode":"image","position":2,"data":{"url":"https://images.example.com/parish-sanctuary.jpg","alt":"The parish sanctuary interior"}},
    {"blockTypeCode":"button","position":3,"data":{"label":"Plan Your Visit","href":"#service-times","style":"primary"}}
  ]'),
('Service times', 'A simple list of weekly service times, ready to edit.', 20,
 '[
    {"blockTypeCode":"heading","position":0,"data":{"level":2,"text":[{"type":"text","text":"Service Times"}]}},
    {"blockTypeCode":"list","position":1,"data":{"content":[{"type":"list","style":"bullet","items":[
      {"type":"listItem","content":[{"type":"text","text":"Sunday Divine Liturgy — 9:00 AM"}]},
      {"type":"listItem","content":[{"type":"text","text":"Saturday Vespers — 5:00 PM"}]}
    ]}]}}
  ]'),
('Meet the clergy', 'Introduce a priest or deacon with a photo, title, and short bio.', 30,
 '[
    {"blockTypeCode":"heading","position":0,"data":{"level":2,"text":[{"type":"text","text":"Meet Our Clergy"}]}},
    {"blockTypeCode":"staff_card","position":1,"data":{"name":"Fr. John Doe","title":"Parish Priest","bio":[{"type":"text","text":"Fr. John has served this parish since 2015, and is available for confession, counsel, and home visits."}]}}
  ]'),
('Getting here', 'Directions and a call to action linking to a map.', 40,
 '[
    {"blockTypeCode":"heading","position":0,"data":{"level":2,"text":[{"type":"text","text":"Getting Here"}]}},
    {"blockTypeCode":"paragraph","position":1,"data":{"text":[{"type":"text","text":"We are located in the heart of town, with parking available on-site. Enter through the main doors facing the street."}]}},
    {"blockTypeCode":"button","position":2,"data":{"label":"Get Directions","href":"https://maps.example.com/","style":"secondary"}}
  ]'),
('Feast-day announcement', 'A short announcement post for an upcoming feast day.', 50,
 '[
    {"blockTypeCode":"heading","position":0,"data":{"level":2,"text":[{"type":"text","text":"Feast Day Announcement"}]}},
    {"blockTypeCode":"paragraph","position":1,"data":{"text":[{"type":"text","text":"Join us as we celebrate the upcoming feast day with a special Divine Liturgy, followed by a parish luncheon. All are welcome."}]}}
  ]');
