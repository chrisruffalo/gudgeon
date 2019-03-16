-- add new "Consumer" column to table
ALTER TABLE qlog ADD Consumer TEXT DEFAULT '';

-- to prevent issues with nil/null columns, set all values to empty
UPDATE qlog SET Consumer = '';