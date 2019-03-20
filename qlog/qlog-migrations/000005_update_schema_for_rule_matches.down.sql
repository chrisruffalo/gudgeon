-- drop indexes for table
DROP INDEX idx_qlog_Address;
DROP INDEX idx_qlog_RequestDomain;
DROP INDEX idx_qlog_Match;
DROP INDEX idx_qlog_Created;
DROP INDEX idx_qlog_Cached;

-- move existing qlog table
ALTER TABLE qlog RENAME to _qlog_old;

-- recreate qlog table
CREATE TABLE qlog (
    Address       TEXT,
    Consumer      TEXT,
    ClientName    TEXT,
    RequestDomain TEXT,
    RequestType   TEXT,
    ResponseText  TEXT,
    Cached        BOOLEAN DEFAULT false,
    Blocked       BOOLEAN,
    BlockedList   TEXT,
    BlockedRule   TEXT,
    Created       DATETIME
);

-- move records into table
INSERT INTO qlog (Address, Consumer, ClientName, RequestDomain, RequestType, ResponseText, Cached, Blocked, BlockedList, BlockedRule, Created)
    SELECT Address, Consumer, ClientName, RequestDomain, RequestType, ResponseText, Cached, Match = 1, MatchList, MatchRule, Created
    FROM _qlog_old;

-- drop old table
DROP TABLE _qlog_old;

-- indexes for table after insert/drop
CREATE INDEX qlog_AddressIndex ON qlog (Address);
CREATE INDEX qlog_RequestDomainIndex ON qlog (RequestDomain);
CREATE INDEX qlog_BlockedIndex ON qlog (Blocked);
CREATE INDEX qlog_CreatedIndex ON qlog (Created);
CREATE INDEX qlog_CachedIndex ON qlog (Cached);