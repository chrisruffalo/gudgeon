-- create initial import table to hold all incoming query data
-- so that it can be batched into other tables but each new
-- query, no matter how it will be processed, requires only one insert
-- the MAIN concerns of this table are to:
--   * GET ALL DATA NECESSARY FOR POPULATING OTHER TABLES
--   * ALLOW INSERTIONS TO BE AS QUICK AS POSSIBLE
CREATE TABLE buffer (
    Id             INTEGER       PRIMARY KEY,
    Address        TEXT          DEFAULT '',
    Consumer       TEXT          DEFAULT '',
    ClientName     TEXT          DEFAULT '',
    RequestDomain  TEXT          DEFAULT '',
    RequestType    TEXT          DEFAULT '',
    ResponseText   TEXT          DEFAULT '',
    Cached         BOOLEAN       DEFAULT false,
    Blocked        BOOLEAN       DEFAULT false,
    Match          INT           DEFAULT 0,
    MatchList      TEXT          DEFAULT '',
    MatchListShort TEXT          DEFAULT '',
    MatchRule      TEXT          DEFAULT '',
    Created        DATETIME,
    StartTime      DATETIME,
    EndTime        DATETIME
);

-- create qlog schema with indexes for long-term storage/use
CREATE TABLE qlog (
    Id             INTEGER       PRIMARY KEY,
    Address        TEXT          DEFAULT '',
    Consumer       TEXT          DEFAULT '',
    ClientName     TEXT          DEFAULT '',
    RequestDomain  TEXT          DEFAULT '',
    RequestType    TEXT          DEFAULT '',
    ResponseText   TEXT          DEFAULT '',
    Cached         BOOLEAN       DEFAULT false,
    Blocked        BOOLEAN       DEFAULT false,
    Match          INT           DEFAULT 0,
    MatchList      TEXT          DEFAULT '',
    MatchListShort TEXT          DEFAULT '',
    MatchRule      TEXT          DEFAULT '',
    Created        DATETIME,
    StartTime      DATETIME,
    EndTime        DATETIME
);
-- create qlog index columns
CREATE INDEX idx_qlog_Address ON qlog (Address);
CREATE INDEX idx_qlog_RequestDomain ON qlog (RequestDomain);
CREATE INDEX idx_qlog_Match ON qlog (Match);
CREATE INDEX idx_qlog_Created ON qlog (Created);
CREATE INDEX idx_qlog_Cached ON qlog (Cached);

-- table for storing periodically collected metrics values
CREATE TABLE metrics (
    FromTime         DATETIME,
    AtTime           DATETIME,
    MetricsJson      TEXT,
    IntervalSeconds  INT
);
-- indexes for table
CREATE INDEX idx_metrics_FromTime ON metrics (FromTime);
CREATE INDEX idx_metrics_AtTime ON metrics (AtTime);

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
CREATE TABLE query_metrics (
    QueryType TEXT PRIMARY KEY DEFAULT '',
    Count INT
) WITHOUT ROWID;
CREATE INDEX idx_query_metrics_QueryType on query_metrics (QueryType);

-- create table for storing individual list metrics
CREATE TABLE list_metrics (
    Id INTEGER PRIMARY KEY,
    Name TEXT DEFAULT '',
    ShortName TEXT DEFAULT '' UNIQUE,
    Hits INT
);
CREATE INDEX idx_list_metrics_ShortName on list_metrics (ShortName);

-- create table for storing rule hits
CREATE TABLE rule_metrics (
    ListId INTEGER,
    Rule TEXT DEFAULT '',
    Hits INT,
    PRIMARY KEY (ListId, Rule)
) WITHOUT ROWID;
CREATE INDEX idx_rule_metrics_ListIdRule on rule_metrics (ListId, Rule);

