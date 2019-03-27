-- nuke buffer and remake
DROP TABLE buffer;
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

-- move old qlog table
ALTER TABLE qlog RENAME TO _qlog_old;

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

-- move records
INSERT INTO qlog (Address, Consumer, ClientName, RequestDomain, RequestType, ResponseText, Cached, Blocked, Match, MatchList, MatchListShort, MatchRule, Created, StartTime, EndTime)
    SELECT Address, Consumer, ClientName, RequestDomain, RequestType, ResponseText, Cached, Blocked, Match, MatchList, MatchListShort, MatchRule, Created, StartTime, EndTime
    FROM _qlog_old;

-- drop old table
DROP TABLE _qlog_old;
