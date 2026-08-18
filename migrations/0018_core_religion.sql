-- 0018_core_religion — M10.1 (D-CorePortScope's amendment: the explicit 15-name list). Ports the
-- kept slice of ../go-oikumenea/migrations/0008_religion.sql. Confirmed this session: grepped this
-- repo (not go-oikumenea) for any reference to religion_unit_classifications — zero matches — so it
-- is dropped, along with the 4 clergy tables and 2 affiliation tables D-CorePortScope always named.
-- religion_taxon_ranks is kept even though the amendment's prose sometimes drops it in headline
-- counts: religion_taxa.rank_id is a NOT NULL FK to it, so omitting it breaks this file outright.
--
-- No Postgres RLS (D-InProcessAuthz). The taxonomy seed (waves 0-4 + closure + theism tags) is
-- ported verbatim from upstream's curated, Wikidata-anchored data (deploy/religion-presets) — static
-- reference data, not project-specific, and this repo's own D-Exclusions taxon codes
-- (russian_orthodox_church / jehovahs_witnesses / lds_church — internal/registration/domain)
-- depend on exact code matches within it.

-- ===================================================================================================
-- Catalogs
-- ===================================================================================================

CREATE TABLE openfaithmap.religion_taxon_ranks (
  id         uuid PRIMARY KEY DEFAULT openfaithmap.new_id(4,1,2),  -- religion / object / taxon_rank
  code       text NOT NULL,
  name       text NOT NULL,
  ordinal    integer NOT NULL,
  status     text NOT NULL DEFAULT 'active' CHECK (status IN ('active','retired')),
  sort_order integer,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  CONSTRAINT religion_taxon_ranks_rid_shape
    CHECK (openfaithmap.rid_service(id)=4 AND openfaithmap.rid_kind(id)=1 AND openfaithmap.rid_type(id)=2)
);
CREATE UNIQUE INDEX religion_taxon_ranks_code_active
  ON openfaithmap.religion_taxon_ranks (code) WHERE deleted_at IS NULL;
CREATE TRIGGER religion_taxon_ranks_set_updated_at
  BEFORE UPDATE ON openfaithmap.religion_taxon_ranks
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();

INSERT INTO openfaithmap.religion_taxon_ranks (code, name, ordinal, sort_order) VALUES
  ('religion','Religion',0,0),
  ('branch','Branch',1,1),
  ('tradition','Tradition',2,2),
  ('sub_tradition','Sub-tradition',3,3),
  ('denomination','Denomination',4,4);

CREATE TABLE openfaithmap.religion_classifications (
  id          uuid PRIMARY KEY DEFAULT openfaithmap.new_id(4,1,3),  -- religion / object / classification
  code        text NOT NULL,
  name        text NOT NULL,
  description text,
  status      text NOT NULL DEFAULT 'active' CHECK (status IN ('active','retired')),
  sort_order  integer,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now(),
  deleted_at  timestamptz,
  CONSTRAINT religion_classifications_rid_shape
    CHECK (openfaithmap.rid_service(id)=4 AND openfaithmap.rid_kind(id)=1 AND openfaithmap.rid_type(id)=3)
);
CREATE UNIQUE INDEX religion_classifications_code_active
  ON openfaithmap.religion_classifications (code) WHERE deleted_at IS NULL;
CREATE TRIGGER religion_classifications_set_updated_at
  BEFORE UPDATE ON openfaithmap.religion_classifications
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();

INSERT INTO openfaithmap.religion_classifications (code, name, sort_order) VALUES
  ('monotheistic','Monotheistic',10),
  ('polytheistic','Polytheistic',20),
  ('henotheistic','Henotheistic',30),
  ('monistic','Monistic',40),
  ('nontheistic','Nontheistic',50),
  ('pantheistic','Pantheistic',60),
  ('panentheistic','Panentheistic',70),
  ('animistic','Animistic',80),
  ('dualistic','Dualistic',90),
  ('deistic','Deistic',100),
  ('agnostic','Agnostic',110),
  ('atheistic','Atheistic',120);

-- ===================================================================================================
-- Taxonomy: the recursive religion_taxa tree + its maintained closure + theism tags.
-- ===================================================================================================

CREATE TABLE openfaithmap.religion_taxa (
  id             uuid PRIMARY KEY DEFAULT openfaithmap.new_id(4,1,1),  -- religion / object / taxon
  parent_id      uuid REFERENCES openfaithmap.religion_taxa(id) ON DELETE RESTRICT,  -- NULL = root religion
  rank_id        uuid NOT NULL REFERENCES openfaithmap.religion_taxon_ranks(id) ON DELETE RESTRICT,
  religion_id    uuid REFERENCES openfaithmap.religion_taxa(id) ON DELETE RESTRICT,  -- denormalized root (derived)
  code           text NOT NULL,
  name           text NOT NULL,
  description    text,
  wikidata_id    text,
  icon           text,
  sort_order     integer,
  source         text,
  source_version text,
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now(),
  deleted_at     timestamptz,
  CONSTRAINT religion_taxa_rid_shape
    CHECK (openfaithmap.rid_service(id)=4 AND openfaithmap.rid_kind(id)=1 AND openfaithmap.rid_type(id)=1),
  CONSTRAINT religion_taxa_not_self_parent CHECK (parent_id IS NULL OR parent_id <> id)
);
CREATE UNIQUE INDEX religion_taxa_code_active
  ON openfaithmap.religion_taxa (code) WHERE deleted_at IS NULL;
CREATE INDEX religion_taxa_parent_idx ON openfaithmap.religion_taxa (parent_id);
CREATE INDEX religion_taxa_rank_idx ON openfaithmap.religion_taxa (rank_id) WHERE deleted_at IS NULL;
CREATE INDEX religion_taxa_religion_idx ON openfaithmap.religion_taxa (religion_id) WHERE deleted_at IS NULL;
CREATE TRIGGER religion_taxa_set_updated_at
  BEFORE UPDATE ON openfaithmap.religion_taxa
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();

CREATE TABLE openfaithmap.religion_taxa_closure (
  ancestor_id   uuid NOT NULL REFERENCES openfaithmap.religion_taxa(id) ON DELETE CASCADE,
  descendant_id uuid NOT NULL REFERENCES openfaithmap.religion_taxa(id) ON DELETE CASCADE,
  depth         integer NOT NULL,
  PRIMARY KEY (ancestor_id, descendant_id)
);
CREATE INDEX religion_taxa_closure_descendant_idx ON openfaithmap.religion_taxa_closure (descendant_id);

CREATE TABLE openfaithmap.religion_taxon_classifications (
  taxon_id          uuid NOT NULL REFERENCES openfaithmap.religion_taxa(id) ON DELETE CASCADE,
  classification_id uuid NOT NULL REFERENCES openfaithmap.religion_classifications(id) ON DELETE RESTRICT,
  created_at        timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (taxon_id, classification_id)
);

-- Curated taxonomy seed (Wikidata-anchored, ported verbatim from deploy/religion-presets). Inserted
-- in waves by level so each row's parent already exists; codes are globally unique so parents
-- resolve by code.

-- Wave 0 — religions (root; parent NULL).
INSERT INTO openfaithmap.religion_taxa (code, name, rank_id, wikidata_id, sort_order, source, source_version)
SELECT v.code, v.name, r.id, v.qid, v.so, 'religion-presets', '2026.06'
FROM (VALUES
  ('christianity','Christianity','Q5043',10),
  ('islam','Islam','Q432',20),
  ('judaism','Judaism','Q9268',30),
  ('hinduism','Hinduism','Q9089',40),
  ('buddhism','Buddhism','Q748',50),
  ('sikhism','Sikhism','Q9554',60),
  ('jainism','Jainism','Q9163',70),
  ('bahai','Bahá''í Faith','Q43066',80),
  ('shinto','Shinto','Q1409',90),
  ('taoism','Taoism','Q9598',100),
  ('confucianism','Confucianism','Q9581',110),
  ('zoroastrianism','Zoroastrianism','Q9601',120),
  ('atheism','Atheism','Q7066',124),
  ('agnosticism','Agnosticism','Q41719',126),
  ('traditional','Traditional & indigenous religions',NULL,130),
  ('other','Other / unclassified',NULL,140)
) AS v(code,name,qid,so)
JOIN openfaithmap.religion_taxon_ranks r ON r.code='religion';

-- Wave 1 — branches (parent = a religion).
INSERT INTO openfaithmap.religion_taxa (code, name, rank_id, parent_id, wikidata_id, sort_order, source, source_version)
SELECT v.code, v.name, rk.id, p.id, v.qid, v.so, 'religion-presets', '2026.06'
FROM (VALUES
  ('christianity','catholicism','Catholicism','Q1841',10),
  ('christianity','eastern_orthodoxy','Eastern Orthodoxy','Q3333484',20),
  ('christianity','oriental_orthodoxy','Oriental Orthodoxy','Q156954',30),
  ('christianity','church_of_the_east','Church of the East','Q470330',40),
  ('christianity','protestantism','Protestantism','Q23540',50),
  ('christianity','restorationism','Restorationism & nontrinitarian','Q1140229',60),
  ('christianity','independent_christianity','Independent / nondenominational',NULL,70),
  ('islam','sunni','Sunni Islam','Q7444',10),
  ('islam','shia','Shia Islam','Q9585',20),
  ('islam','ibadi','Ibadi Islam','Q319540',30),
  ('islam','sufism','Sufism','Q9622',40),
  ('islam','ahmadiyya','Ahmadiyya','Q170027',50),
  ('judaism','orthodox_judaism','Orthodox Judaism','Q170238',10),
  ('judaism','conservative_judaism','Conservative Judaism','Q188476',20),
  ('judaism','reform_judaism','Reform Judaism','Q102045',30),
  ('judaism','reconstructionist_judaism','Reconstructionist Judaism','Q1150985',40),
  ('judaism','karaite_judaism','Karaite Judaism','Q484591',50),
  ('hinduism','vaishnavism','Vaishnavism','Q842337',10),
  ('hinduism','shaivism','Shaivism','Q319183',20),
  ('hinduism','shaktism','Shaktism','Q1132099',30),
  ('hinduism','smartism','Smartism','Q707348',40),
  ('buddhism','theravada','Theravāda','Q49003',10),
  ('buddhism','mahayana','Mahāyāna','Q43361',20),
  ('buddhism','vajrayana','Vajrayāna','Q489704',30),
  ('jainism','digambara','Digambara','Q1189537',10),
  ('jainism','svetambara','Śvetāmbara','Q726177',20)
) AS v(parent_code,code,name,qid,so)
JOIN openfaithmap.religion_taxon_ranks rk ON rk.code='branch'
JOIN openfaithmap.religion_taxa p ON p.code=v.parent_code AND p.deleted_at IS NULL;

-- Wave 2 — traditions / movements (parent = a branch).
INSERT INTO openfaithmap.religion_taxa (code, name, rank_id, parent_id, wikidata_id, sort_order, source, source_version)
SELECT v.code, v.name, rk.id, p.id, v.qid, v.so, 'religion-presets', '2026.06'
FROM (VALUES
  ('catholicism','latin_church','Latin Church','Q612330',10),
  ('catholicism','eastern_catholic','Eastern Catholic Churches','Q751392',20),
  ('protestantism','lutheranism','Lutheranism','Q75809',10),
  ('protestantism','reformed','Reformed (Calvinism)','Q101849',20),
  ('protestantism','anglicanism','Anglicanism','Q6423963',30),
  ('protestantism','anabaptism','Anabaptism','Q104088',40),
  ('protestantism','baptist','Baptist','Q93191',50),
  ('protestantism','methodism','Methodism','Q104400',60),
  ('protestantism','pentecostalism','Pentecostalism','Q170022',70),
  ('protestantism','adventism','Adventism','Q164359',80),
  ('protestantism','holiness','Holiness movement','Q1535557',90),
  ('protestantism','evangelicalism','Evangelicalism','Q170997',100),
  ('protestantism','quakerism','Quakerism (Friends)','Q170582',110)
) AS v(parent_code,code,name,qid,so)
JOIN openfaithmap.religion_taxon_ranks rk ON rk.code='tradition'
JOIN openfaithmap.religion_taxa p ON p.code=v.parent_code AND p.deleted_at IS NULL;

-- Wave 3 — sub-traditions (rite / school / madhhab; parent = a tradition or branch).
INSERT INTO openfaithmap.religion_taxa (code, name, rank_id, parent_id, wikidata_id, sort_order, source, source_version)
SELECT v.code, v.name, rk.id, p.id, v.qid, v.so, 'religion-presets', '2026.06'
FROM (VALUES
  ('sunni','hanafi','Hanafi','Q223097',10),
  ('sunni','maliki','Maliki','Q207922',20),
  ('sunni','shafii','Shafiʿi','Q220910',30),
  ('sunni','hanbali','Hanbali','Q200671',40),
  ('shia','twelver','Twelver (Ithnāʿasharī)','Q170382',10),
  ('shia','ismailism','Ismailism','Q179872',20),
  ('shia','zaidiyyah','Zaidiyyah','Q319618',30),
  ('orthodox_judaism','hasidic','Hasidic Judaism','Q170581',10),
  ('orthodox_judaism','modern_orthodox','Modern Orthodox Judaism','Q1426764',20),
  ('orthodox_judaism','haredi','Haredi Judaism','Q208163',30),
  ('reformed','presbyterianism','Presbyterianism','Q178169',10),
  ('reformed','congregationalism','Congregationalism','Q1062789',20),
  ('reformed','continental_reformed','Continental Reformed','Q1129121',30)
) AS v(parent_code,code,name,qid,so)
JOIN openfaithmap.religion_taxon_ranks rk ON rk.code='sub_tradition'
JOIN openfaithmap.religion_taxa p ON p.code=v.parent_code AND p.deleted_at IS NULL;

-- Wave 4 — denominations: the globally-significant historic churches/bodies.
INSERT INTO openfaithmap.religion_taxa (code, name, rank_id, parent_id, wikidata_id, sort_order, source, source_version)
SELECT v.code, v.name, rk.id, p.id, v.qid, v.so, 'religion-presets', '2026.06'
FROM (VALUES
  ('eastern_orthodoxy','ecumenical_patriarchate','Ecumenical Patriarchate of Constantinople','Q656861',10),
  ('eastern_orthodoxy','church_of_greece','Church of Greece','Q732221',20),
  ('eastern_orthodoxy','russian_orthodox_church','Russian Orthodox Church','Q60150',30),
  ('eastern_orthodoxy','serbian_orthodox_church','Serbian Orthodox Church','Q170377',40),
  ('eastern_orthodoxy','romanian_orthodox_church','Romanian Orthodox Church','Q463041',50),
  ('eastern_orthodoxy','bulgarian_orthodox_church','Bulgarian Orthodox Church','Q463848',60),
  ('eastern_orthodoxy','georgian_orthodox_church','Georgian Orthodox Church','Q1129877',70),
  ('eastern_orthodoxy','orthodox_church_of_ukraine','Orthodox Church of Ukraine','Q30901814',80),
  ('eastern_orthodoxy','orthodox_church_in_america','Orthodox Church in America','Q673354',90),
  ('oriental_orthodoxy','coptic_orthodox_church','Coptic Orthodox Church','Q56183',10),
  ('oriental_orthodoxy','armenian_apostolic_church','Armenian Apostolic Church','Q102140',20),
  ('oriental_orthodoxy','ethiopian_orthodox_tewahedo','Ethiopian Orthodox Tewahedo Church','Q260415',30),
  ('oriental_orthodoxy','syriac_orthodox_church','Syriac Orthodox Church','Q464345',40),
  ('oriental_orthodoxy','malankara_orthodox_church','Malankara Orthodox Syrian Church','Q1815695',50),
  ('church_of_the_east','assyrian_church_of_the_east','Assyrian Church of the East','Q178379',10),
  ('church_of_the_east','ancient_church_of_the_east','Ancient Church of the East','Q1130645',20),
  ('eastern_catholic','ukrainian_greek_catholic_church','Ukrainian Greek Catholic Church','Q1192126',10),
  ('eastern_catholic','maronite_church','Maronite Church','Q827512',20),
  ('eastern_catholic','melkite_greek_catholic_church','Melkite Greek Catholic Church','Q1185801',30),
  ('eastern_catholic','chaldean_catholic_church','Chaldean Catholic Church','Q656801',40),
  ('eastern_catholic','syro_malabar_church','Syro-Malabar Church','Q1163901',50),
  ('eastern_catholic','armenian_catholic_church','Armenian Catholic Church','Q807607',60),
  ('lutheranism','elca','Evangelical Lutheran Church in America','Q1340004',10),
  ('lutheranism','lcms','Lutheran Church – Missouri Synod','Q1473773',20),
  ('baptist','southern_baptist_convention','Southern Baptist Convention','Q815672',10),
  ('methodism','united_methodist_church','United Methodist Church','Q1446703',10),
  ('anglicanism','church_of_england','Church of England','Q82708',10),
  ('anglicanism','episcopal_church_usa','Episcopal Church (USA)','Q1366000',20),
  ('pentecostalism','assemblies_of_god','Assemblies of God','Q598397',10),
  ('adventism','seventh_day_adventist_church','Seventh-day Adventist Church','Q104319',10),
  ('restorationism','lds_church','Church of Jesus Christ of Latter-day Saints','Q19595',10),
  ('restorationism','jehovahs_witnesses','Jehovah''s Witnesses','Q35269',20)
) AS v(parent_code,code,name,qid,so)
JOIN openfaithmap.religion_taxon_ranks rk ON rk.code='denomination'
JOIN openfaithmap.religion_taxa p ON p.code=v.parent_code AND p.deleted_at IS NULL;

-- Bulk-build the closure over the seeded tree (reflexive + every ancestor->descendant pair).
INSERT INTO openfaithmap.religion_taxa_closure (ancestor_id, descendant_id, depth)
WITH RECURSIVE anc AS (
  SELECT id AS ancestor_id, id AS descendant_id, 0 AS depth
  FROM openfaithmap.religion_taxa WHERE deleted_at IS NULL
  UNION ALL
  SELECT a.ancestor_id, t.id, a.depth + 1
  FROM anc a
  JOIN openfaithmap.religion_taxa t ON t.parent_id = a.descendant_id AND t.deleted_at IS NULL
)
SELECT ancestor_id, descendant_id, depth FROM anc;

-- Derive each taxon's denormalized root religion_id.
UPDATE openfaithmap.religion_taxa t
SET religion_id = root.ancestor_id
FROM (
  SELECT c.descendant_id, c.ancestor_id
  FROM openfaithmap.religion_taxa_closure c
  JOIN openfaithmap.religion_taxa a ON a.id = c.ancestor_id AND a.parent_id IS NULL
) root
WHERE root.descendant_id = t.id;

-- Theism tags, seeded at the religion level (overridable lower down via the closure resolution).
INSERT INTO openfaithmap.religion_taxon_classifications (taxon_id, classification_id)
SELECT t.id, c.id
FROM (VALUES
  ('christianity','monotheistic'),
  ('islam','monotheistic'),
  ('judaism','monotheistic'),
  ('sikhism','monotheistic'),
  ('bahai','monotheistic'),
  ('zoroastrianism','monotheistic'),
  ('zoroastrianism','dualistic'),
  ('hinduism','monotheistic'),
  ('hinduism','polytheistic'),
  ('hinduism','henotheistic'),
  ('hinduism','monistic'),
  ('buddhism','nontheistic'),
  ('jainism','nontheistic'),
  ('jainism','polytheistic'),
  ('shinto','polytheistic'),
  ('shinto','animistic'),
  ('taoism','pantheistic'),
  ('taoism','polytheistic'),
  ('confucianism','nontheistic'),
  ('atheism','atheistic'),
  ('agnosticism','agnostic'),
  ('traditional','animistic'),
  ('traditional','polytheistic')
) AS v(taxon_code, class_code)
JOIN openfaithmap.religion_taxa t ON t.code=v.taxon_code AND t.deleted_at IS NULL
JOIN openfaithmap.religion_classifications c ON c.code=v.class_code;

-- ===================================================================================================
-- Organization catalogs.
-- ===================================================================================================

CREATE TABLE openfaithmap.religion_org_kinds (
  id          uuid PRIMARY KEY DEFAULT openfaithmap.new_id(4,1,4),  -- religion / object / org_kind
  religion_id uuid REFERENCES openfaithmap.religion_taxa(id) ON DELETE RESTRICT,  -- NULL = generic
  code        text NOT NULL,
  name        text NOT NULL,
  ordinal     integer,
  status      text NOT NULL DEFAULT 'active' CHECK (status IN ('active','retired')),
  sort_order  integer,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now(),
  deleted_at  timestamptz,
  CONSTRAINT religion_org_kinds_rid_shape
    CHECK (openfaithmap.rid_service(id)=4 AND openfaithmap.rid_kind(id)=1 AND openfaithmap.rid_type(id)=4)
);
CREATE UNIQUE INDEX religion_org_kinds_code_active
  ON openfaithmap.religion_org_kinds (code) WHERE deleted_at IS NULL;
CREATE TRIGGER religion_org_kinds_set_updated_at
  BEFORE UPDATE ON openfaithmap.religion_org_kinds
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();

INSERT INTO openfaithmap.religion_org_kinds (code, name, ordinal, sort_order) VALUES
  ('denomination','Denomination',10,10),
  ('jurisdiction','Jurisdiction',20,20),
  ('diocese','Diocese / Eparchy',30,30),
  ('deanery','Deanery',40,40),
  ('parish','Parish',50,50),
  ('congregation','Congregation',60,60),
  ('mission','Mission',70,70),
  ('monastery','Monastery',80,80),
  ('community','Community',90,90),
  ('mosque_community','Mosque community',100,100),
  ('temple_community','Temple community',110,110),
  ('council','Council / Association',120,120);

CREATE TABLE openfaithmap.religion_policy_kinds (
  id          uuid PRIMARY KEY DEFAULT openfaithmap.new_id(4,1,5),  -- religion / object / policy_kind
  code        text NOT NULL,
  name        text NOT NULL,
  description text,
  status      text NOT NULL DEFAULT 'active' CHECK (status IN ('active','retired')),
  sort_order  integer,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now(),
  deleted_at  timestamptz,
  CONSTRAINT religion_policy_kinds_rid_shape
    CHECK (openfaithmap.rid_service(id)=4 AND openfaithmap.rid_kind(id)=1 AND openfaithmap.rid_type(id)=5)
);
CREATE UNIQUE INDEX religion_policy_kinds_code_active
  ON openfaithmap.religion_policy_kinds (code) WHERE deleted_at IS NULL;
CREATE TRIGGER religion_policy_kinds_set_updated_at
  BEFORE UPDATE ON openfaithmap.religion_policy_kinds
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();

INSERT INTO openfaithmap.religion_policy_kinds (code, name, description, sort_order) VALUES
  ('excludes_child_creation','Excludes child creation','Blocks creating organizations beneath this body',10),
  ('excluded_body','Excluded body','Marks this body as excluded/derecognized for a stated reason',20);

-- ===================================================================================================
-- Organization (reuses directory_units): per-unit faith profile + classifications + policies.
-- ===================================================================================================

CREATE TABLE openfaithmap.religion_org_profiles (
  unit_id     uuid PRIMARY KEY REFERENCES openfaithmap.directory_units(id) ON DELETE RESTRICT,
  org_kind_id uuid REFERENCES openfaithmap.religion_org_kinds(id) ON DELETE RESTRICT,
  short_code  text,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now(),
  deleted_at  timestamptz
);
CREATE TRIGGER religion_org_profiles_set_updated_at
  BEFORE UPDATE ON openfaithmap.religion_org_profiles
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();

CREATE TABLE openfaithmap.religion_org_classifications (
  id         uuid PRIMARY KEY DEFAULT openfaithmap.new_id(4,2,1),  -- religion / link / classified_as
  unit_id    uuid NOT NULL REFERENCES openfaithmap.directory_units(id) ON DELETE CASCADE,
  taxon_id   uuid NOT NULL REFERENCES openfaithmap.religion_taxa(id) ON DELETE RESTRICT,
  is_primary boolean NOT NULL DEFAULT false,
  source     text,
  confidence text,
  valid_from timestamptz NOT NULL DEFAULT now(),
  valid_to   timestamptz CHECK (valid_to IS NULL OR valid_to >= valid_from),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  CONSTRAINT religion_org_classifications_rid_shape
    CHECK (openfaithmap.rid_service(id)=4 AND openfaithmap.rid_kind(id)=2 AND openfaithmap.rid_type(id)=1)
);
CREATE UNIQUE INDEX religion_org_classifications_unit_taxon_active
  ON openfaithmap.religion_org_classifications (unit_id, taxon_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX religion_org_classifications_one_primary
  ON openfaithmap.religion_org_classifications (unit_id) WHERE is_primary AND deleted_at IS NULL;
CREATE INDEX religion_org_classifications_taxon_idx
  ON openfaithmap.religion_org_classifications (taxon_id) WHERE deleted_at IS NULL;
CREATE TRIGGER religion_org_classifications_set_updated_at
  BEFORE UPDATE ON openfaithmap.religion_org_classifications
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();

CREATE TABLE openfaithmap.religion_org_policies (
  id                    uuid PRIMARY KEY DEFAULT openfaithmap.new_id(4,1,6),  -- religion / object / org_policy
  unit_id               uuid NOT NULL REFERENCES openfaithmap.directory_units(id) ON DELETE CASCADE,
  policy_kind_id        uuid NOT NULL REFERENCES openfaithmap.religion_policy_kinds(id) ON DELETE RESTRICT,
  reason                text,
  decided_by_person_id  uuid REFERENCES openfaithmap.identity_persons(id) ON DELETE SET NULL,
  decided_at            timestamptz,
  created_at            timestamptz NOT NULL DEFAULT now(),
  updated_at            timestamptz NOT NULL DEFAULT now(),
  deleted_at            timestamptz,
  CONSTRAINT religion_org_policies_rid_shape
    CHECK (openfaithmap.rid_service(id)=4 AND openfaithmap.rid_kind(id)=1 AND openfaithmap.rid_type(id)=6)
);
CREATE UNIQUE INDEX religion_org_policies_unit_kind_active
  ON openfaithmap.religion_org_policies (unit_id, policy_kind_id) WHERE deleted_at IS NULL;
CREATE INDEX religion_org_policies_unit_idx
  ON openfaithmap.religion_org_policies (unit_id) WHERE deleted_at IS NULL;
CREATE TRIGGER religion_org_policies_set_updated_at
  BEFORE UPDATE ON openfaithmap.religion_org_policies
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();

-- ===================================================================================================
-- religion_sites / schedules / aliases — the discovery substrate. religion_sites references
-- location_locations (0019_core_location.sql, applied after this file — see its own header) and
-- religion_site_types (this file); the location FK is added there, not here, since location doesn't
-- exist yet at this point in the migration sequence.
-- ===================================================================================================

CREATE TABLE openfaithmap.religion_site_types (
  id                 uuid PRIMARY KEY DEFAULT openfaithmap.new_id(4,1,7),  -- religion / object / site_type
  tradition_taxon_id uuid REFERENCES openfaithmap.religion_taxa(id) ON DELETE RESTRICT,  -- NULL = generic
  code               text NOT NULL,
  name               text NOT NULL,
  status             text NOT NULL DEFAULT 'active' CHECK (status IN ('active','retired')),
  sort_order         integer,
  created_at         timestamptz NOT NULL DEFAULT now(),
  updated_at         timestamptz NOT NULL DEFAULT now(),
  deleted_at         timestamptz,
  CONSTRAINT religion_site_types_rid_shape
    CHECK (openfaithmap.rid_service(id)=4 AND openfaithmap.rid_kind(id)=1 AND openfaithmap.rid_type(id)=7)
);
CREATE UNIQUE INDEX religion_site_types_code_active
  ON openfaithmap.religion_site_types (tradition_taxon_id, code) WHERE deleted_at IS NULL;
CREATE TRIGGER religion_site_types_set_updated_at
  BEFORE UPDATE ON openfaithmap.religion_site_types
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();

INSERT INTO openfaithmap.religion_site_types (tradition_taxon_id, code, name, sort_order) VALUES
  (NULL,'office','Office',100),
  (NULL,'online','Online',110),
  (NULL,'mission','Mission',120),
  (NULL,'shrine','Shrine',130);
INSERT INTO openfaithmap.religion_site_types (tradition_taxon_id, code, name, sort_order)
SELECT t.id, v.code, v.name, v.so
FROM (VALUES
  ('christianity','church','Church',10),
  ('christianity','cathedral','Cathedral',20),
  ('christianity','chapel','Chapel',30),
  ('christianity','monastery','Monastery',40),
  ('islam','mosque','Mosque',10),
  ('judaism','synagogue','Synagogue',10),
  ('hinduism','temple','Temple',10),
  ('buddhism','temple','Temple',10),
  ('sikhism','gurdwara','Gurdwara',10)
) AS v(tradition_code, code, name, so)
JOIN openfaithmap.religion_taxa t ON t.code = v.tradition_code AND t.deleted_at IS NULL;

CREATE TABLE openfaithmap.religion_service_types (
  id                 uuid PRIMARY KEY DEFAULT openfaithmap.new_id(4,1,8),  -- religion / object / service_type
  tradition_taxon_id uuid REFERENCES openfaithmap.religion_taxa(id) ON DELETE RESTRICT,  -- NULL = generic
  code               text NOT NULL,
  name               text NOT NULL,
  status             text NOT NULL DEFAULT 'active' CHECK (status IN ('active','retired')),
  sort_order         integer,
  created_at         timestamptz NOT NULL DEFAULT now(),
  updated_at         timestamptz NOT NULL DEFAULT now(),
  deleted_at         timestamptz,
  CONSTRAINT religion_service_types_rid_shape
    CHECK (openfaithmap.rid_service(id)=4 AND openfaithmap.rid_kind(id)=1 AND openfaithmap.rid_type(id)=8)
);
CREATE UNIQUE INDEX religion_service_types_code_active
  ON openfaithmap.religion_service_types (tradition_taxon_id, code) WHERE deleted_at IS NULL;
CREATE TRIGGER religion_service_types_set_updated_at
  BEFORE UPDATE ON openfaithmap.religion_service_types
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();

INSERT INTO openfaithmap.religion_service_types (tradition_taxon_id, code, name, sort_order) VALUES
  (NULL,'main','Main service',10),
  (NULL,'youth','Youth service',20),
  (NULL,'prayer','Prayer',30),
  (NULL,'special','Special service',40);
INSERT INTO openfaithmap.religion_service_types (tradition_taxon_id, code, name, sort_order)
SELECT t.id, v.code, v.name, v.so
FROM (VALUES
  ('christianity','daily_mass','Daily Mass',50),
  ('islam','jumua','Friday (Jumuʿah) prayer',50),
  ('judaism','shabbat','Shabbat service',50),
  ('hinduism','puja','Puja',50),
  ('buddhism','meditation','Meditation',50)
) AS v(tradition_code, code, name, so)
JOIN openfaithmap.religion_taxa t ON t.code = v.tradition_code AND t.deleted_at IS NULL;

-- religion_sites: the reified Unit<->Location link. location_id's FK is added in
-- 0019_core_location.sql once location_locations exists.
CREATE TABLE openfaithmap.religion_sites (
  id               uuid PRIMARY KEY DEFAULT openfaithmap.new_id(4,2,2),  -- religion / link / site_of
  org_unit_id      uuid NOT NULL REFERENCES openfaithmap.directory_units(id) ON DELETE RESTRICT,
  location_id      uuid NOT NULL,  -- REFERENCES openfaithmap.location_locations(id); FK added in 0019_core_location.sql
  site_type_id     uuid NOT NULL REFERENCES openfaithmap.religion_site_types(id) ON DELETE RESTRICT,
  visibility       text NOT NULL DEFAULT 'public' CHECK (visibility IN ('public','unlisted','private')),
  public_precision text NOT NULL DEFAULT 'exact'
                     CHECK (public_precision IN ('exact','street','neighborhood','city','hidden')),
  is_primary       boolean NOT NULL DEFAULT false,
  valid_from       timestamptz NOT NULL DEFAULT now(),
  valid_to         timestamptz CHECK (valid_to IS NULL OR valid_to >= valid_from),
  created_at       timestamptz NOT NULL DEFAULT now(),
  updated_at       timestamptz NOT NULL DEFAULT now(),
  deleted_at       timestamptz,
  CONSTRAINT religion_sites_rid_shape
    CHECK (openfaithmap.rid_service(id)=4 AND openfaithmap.rid_kind(id)=2 AND openfaithmap.rid_type(id)=2)
);
CREATE UNIQUE INDEX religion_sites_one_primary
  ON openfaithmap.religion_sites (org_unit_id) WHERE is_primary AND deleted_at IS NULL;
CREATE INDEX religion_sites_unit_idx ON openfaithmap.religion_sites (org_unit_id) WHERE deleted_at IS NULL;
CREATE TRIGGER religion_sites_set_updated_at
  BEFORE UPDATE ON openfaithmap.religion_sites
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();

CREATE TABLE openfaithmap.religion_service_schedules (
  id              uuid PRIMARY KEY DEFAULT openfaithmap.new_id(4,1,9),  -- religion / object / service_schedule
  site_id         uuid NOT NULL REFERENCES openfaithmap.religion_sites(id) ON DELETE CASCADE,
  service_type_id uuid NOT NULL REFERENCES openfaithmap.religion_service_types(id) ON DELETE RESTRICT,
  day_of_week     smallint CHECK (day_of_week BETWEEN 0 AND 6),
  rrule           text,
  start_time      time,
  end_time        time,
  timezone        text NOT NULL,
  language        text,
  mode            text NOT NULL DEFAULT 'in_person' CHECK (mode IN ('in_person','online','hybrid')),
  meeting_url     text,
  description     text,
  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now(),
  deleted_at      timestamptz,
  CONSTRAINT religion_service_schedules_rid_shape
    CHECK (openfaithmap.rid_service(id)=4 AND openfaithmap.rid_kind(id)=1 AND openfaithmap.rid_type(id)=9),
  CONSTRAINT religion_service_schedules_recurrence CHECK (day_of_week IS NOT NULL OR rrule IS NOT NULL)
);
CREATE INDEX religion_service_schedules_site_idx
  ON openfaithmap.religion_service_schedules (site_id) WHERE deleted_at IS NULL;
CREATE TRIGGER religion_service_schedules_set_updated_at
  BEFORE UPDATE ON openfaithmap.religion_service_schedules
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();

CREATE TABLE openfaithmap.religion_aliases (
  id         uuid PRIMARY KEY DEFAULT openfaithmap.new_id(4,1,10),  -- religion / object / alias
  unit_id    uuid NOT NULL REFERENCES openfaithmap.directory_units(id) ON DELETE CASCADE,
  alias_text text NOT NULL,
  alias_type text NOT NULL
             CHECK (alias_type IN ('nickname','abbreviation','historical','misspelling','transliteration')),
  locale     text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  CONSTRAINT religion_aliases_rid_shape
    CHECK (openfaithmap.rid_service(id)=4 AND openfaithmap.rid_kind(id)=1 AND openfaithmap.rid_type(id)=10)
);
CREATE INDEX religion_aliases_unit_idx ON openfaithmap.religion_aliases (unit_id) WHERE deleted_at IS NULL;
CREATE INDEX religion_aliases_text_idx ON openfaithmap.religion_aliases (lower(alias_text)) WHERE deleted_at IS NULL;
CREATE TRIGGER religion_aliases_set_updated_at
  BEFORE UPDATE ON openfaithmap.religion_aliases
  FOR EACH ROW EXECUTE FUNCTION openfaithmap.set_updated_at();
