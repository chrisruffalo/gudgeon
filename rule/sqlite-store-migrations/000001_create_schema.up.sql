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
-- without these indexes it is **much** slower but these are implemented in code after inserts
-- because it is faster to create the inserts there but they are recorded here for clarity on
-- what indexes will be created
-- CREATE INDEX IF NOT EXISTS idx_rules_Rule on rules (Rule);
-- CREATE INDEX IF NOT EXISTS idx_rules_ListRowId on rules (ListRowId);
-- CREATE INDEX IF NOT EXISTS idx_rules_IdRule on rules (ListRowId, Rule);