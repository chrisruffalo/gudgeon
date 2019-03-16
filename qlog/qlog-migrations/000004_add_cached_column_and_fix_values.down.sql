-- drop indexes for table
DROP INDEX qlog_AddressIndex;
DROP INDEX qlog_RequestDomainIndex;
DROP INDEX qlog_BlockedIndex;
DROP INDEX qlog_CreatedIndex;

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
    Blocked       BOOLEAN,
    BlockedList   TEXT,
    BlockedRule   TEXT,
    Created       DATETIME
);

-- move records into table
INSERT INTO qlog (Address, Consumer, ClientName, RequestDomain, RequestType, ResponseText, Blocked, BlockedList, BlockedRule, Created)
    SELECT Address, Consumer, ClientName, RequestDomain, RequestType, ResponseText, Blocked, BlockedList, BlockedRule, Created
    FROM _qlog_old;

-- drop old table
DROP TABLE _qlog_old;

-- fix earlier issues with migrations 02 and 03
UPDATE qlog SET Consumer = null where Consumer = '';
UPDATE qlog SET ClientName = null where ClientName = '';

-- indexes for table after insert/drop
CREATE INDEX qlog_AddressIndex ON qlog (Address);
CREATE INDEX qlog_RequestDomainIndex ON qlog (RequestDomain);
CREATE INDEX qlog_BlockedIndex ON qlog (Blocked);
CREATE INDEX qlog_CreatedIndex ON qlog (Created);