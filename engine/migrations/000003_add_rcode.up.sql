-- add column for rcode
ALTER TABLE buffer ADD COLUMN Rcode TEXT DEFAULT '';
UPDATE buffer SET RCODE = '' WHERE RCODE = null;

-- add query log column for rcode
ALTER TABLE qlog ADD COLUMN Rcode TEXT DEFAULT '';
UPDATE qlog SET RCODE = '' WHERE RCODE = null;

