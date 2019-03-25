-- shadow table for inserts with batch move into qlog table
CREATE TABLE qlog_temp (
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
