-- move existing qlog table
ALTER TABLE qlog RENAME to _qlog_old;

-- recreate qlog table
CREATE TABLE qlog (
    Address       TEXT,
    RequestDomain TEXT,
    RequestType   TEXT,
    ResponseText  TEXT,
    Blocked       BOOLEAN,
    BlockedList   TEXT,
    BlockedRule   TEXT,
    Created       DATETIME
);
-- indexes for table
CREATE INDEX qlog_AddressIndex ON qlog (Address);
CREATE INDEX qlog_RequestDomainIndex ON qlog (RequestDomain);
CREATE INDEX qlog_BlockedIndex ON qlog (Blocked);
CREATE INDEX qlog_CreatedIndex ON qlog (Created);

-- move records into table
INSERT INTO qlog (Address, RequestDomain, RequestType, ResponseText, Blocked, BlockedList, BlockedRule, Created)
SELECT Address, RequestDomain, RequestType, ResponseText, Blocked, BlockedList, BlockedRule, Created
FROM _qlog_old;

-- drop old table
DROP TABLE qlog_old;