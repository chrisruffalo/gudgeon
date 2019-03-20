-- create new table for storing client metrics
CREATE TABLE client_metrics (
    Address TEXT PRIMARY KEY DEFAULT '',
    Count INT
) WITHOUT ROWID;
CREATE INDEX idx_client_metrics_Address ON client_metrics (Address);

-- create new table for storing domain metrics
CREATE TABLE domain_metrics (
    DomainName TEXT PRIMARY KEY DEFAULT '',
    Count INT
) WITHOUT ROWID;
CREATE INDEX idx_domain_metrics_DomainName on domain_metrics (DomainName);

-- create table for storing query type metrics
CREATE TABLE query_types (
    QueryType TEXT PRIMARY KEY DEFAULT '',
    Count INT
) WITHOUT ROWID;
CREATE INDEX idx_query_types_QueryType on query_types (QueryType);

-- create new tables for storing rule-specific metrics
CREATE TABLE lists (
    Id INTEGER PRIMARY KEY,
    Name TEXT DEFAULT '',
    ShortName TEXT DEFAULT '',
    Hits INT
);
CREATE INDEX idx_lists_ShortName on lists (ShortName);

CREATE TABLE rule_hits (
    ListId INTEGER,
    Rule TEXT DEFAULT '',
    Hits INT,
    PRIMARY KEY (ListId, Rule)
) WITHOUT ROWID;
CREATE INDEX idx_rule_hits_ListId on rule_hits (ListId);
CREATE INDEX idx_rule_hits_Rule on rule_hits (Rule);
CREATE INDEX idx_rule_hits_ListIdRule on rule_hits (ListId, Rule);