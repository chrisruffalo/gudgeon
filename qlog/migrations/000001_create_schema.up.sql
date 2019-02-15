BEGIN TRANSACTION;
    CREATE TABLE qlog (
        Address       string,
        RequestDomain string,
        RequestType   string,
        ResponseText  string,
        Blocked       bool,
        BlockedList   string,
        BlockedRule   string,
        Created       time
    );
    CREATE INDEX qlog_AddressIndex ON qlog (Address);
    CREATE INDEX qlog_RequestDomainIndex ON qlog (RequestDomain);
    CREATE INDEX qlog_BlockedIndex ON qlog (Blocked);
    CREATE INDEX qlog_CreatedIndex ON qlog (Created);
COMMIT;
