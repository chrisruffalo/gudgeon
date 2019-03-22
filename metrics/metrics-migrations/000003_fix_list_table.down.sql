-- drop and recreate table
DROP TABLE lists;

-- restore old non-unique table
CREATE TABLE lists (
    Id INTEGER PRIMARY KEY,
    Name TEXT DEFAULT '',
    ShortName TEXT DEFAULT '',
    Hits INT
);
CREATE INDEX idx_lists_ShortName on lists (ShortName);