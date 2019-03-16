-- add new "Cached" column to table
ALTER TABLE qlog ADD Cached BOOLEAN DEFAULT false;

-- to prevent issues with nil/null columns, set all values to false
UPDATE qlog SET Cached = false;

-- fix earlier issues with migrations 02 and 03
UPDATE qlog SET Consumer = '' where Consumer = null;
UPDATE qlog SET ClientName = '' where ClientName = null;

-- add new index column
CREATE INDEX qlog_CachedIndex ON qlog (Cached);