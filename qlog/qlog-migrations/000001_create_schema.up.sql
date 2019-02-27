-- table structure
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
