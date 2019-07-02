-- no automatic indexing
PRAGMA automatic_index = false;

-- list table exists so that we don't have to store the entire list name for each rule entry
CREATE TABLE IF NOT EXISTS lists ( 
    Id INTEGER PRIMARY KEY, 
    ShortName TEXT 
);
CREATE INDEX IF NOT EXISTS idx_lists_ShortName on lists (ShortName);

-- rules table is joined against the list table for unique sets of rules
CREATE TABLE IF NOT EXISTS rules ( 
    ListRowId INTEGER, 
    Rule TEXT, 
    PRIMARY KEY(ListRowId, Rule) 
);
-- without this index it is **much** slower
CREATE INDEX IF NOT EXISTS idx_rules_Rule on rules (Rule);
