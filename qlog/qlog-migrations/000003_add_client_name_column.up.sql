-- add new "ClientName" column to table
ALTER TABLE qlog ADD ClientName TEXT DEFAULT '';

-- to prevent issues with nil/null columns, set all values to empty
UPDATE qlog SET ClientName = '';