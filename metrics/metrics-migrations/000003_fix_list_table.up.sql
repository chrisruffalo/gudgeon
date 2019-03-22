-- drop and recreate table
DROP TABLE lists;

-- short name must be unique
CREATE TABLE lists (
    Id INTEGER PRIMARY KEY,
    Name TEXT DEFAULT '',
    ShortName TEXT DEFAULT '' UNIQUE,
    Hits INT
);
CREATE INDEX idx_lists_ShortName on lists (ShortName);