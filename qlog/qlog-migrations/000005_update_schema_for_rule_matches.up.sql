-- move existing qlog table
ALTER TABLE qlog RENAME to _qlog_old;

-- recreate qlog schema but for hits
-- blocked is now a HARD BLOCK at the consumer level
CREATE TABLE qlog (
    Id            INTEGER  PRIMARY KEY,
    Address       TEXT     DEFAULT '',
    Consumer      TEXT     DEFAULT '',
    ClientName    TEXT     DEFAULT '',
    RequestDomain TEXT     DEFAULT '',
    RequestType   TEXT     DEFAULT '',
    ResponseText  TEXT     DEFAULT '',
    Cached        BOOLEAN  DEFAULT false,
    Blocked       BOOLEAN  DEFAULT false,
    Match         INT      DEFAULT 0,
    MatchList     TEXT     DEFAULT '',
    MatchRule     TEXT     DEFAULT '',
    Created       DATETIME
);

-- create index columns
CREATE INDEX idx_qlog_Address ON qlog (Address);
CREATE INDEX idx_qlog_RequestDomain ON qlog (RequestDomain);
CREATE INDEX idx_qlog_Match ON qlog (Match);
CREATE INDEX idx_qlog_Created ON qlog (Created);
CREATE INDEX idx_qlog_Cached ON qlog (Cached);

-- move values into new table
INSERT INTO qlog (Id, Address, Consumer, ClientName, RequestDomain, RequestType, ResponseText, Cached, MatchList, MatchRule, Created)
    SELECT rowid, Address, Consumer, ClientName, RequestDomain, RequestType, ResponseText, Cached, BlockedList, BlockedRule, Created
    FROM _qlog_old;

-- set match value to rule.MatchBlocked ( = 1) from old blocked column
UPDATE qlog SET Match = 1 WHERE Id in (SELECT rowid FROM _qlog_old WHERE Blocked = true);

-- drop old table
DROP TABLE _qlog_old;