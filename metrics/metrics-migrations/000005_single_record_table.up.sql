-- create table that all metrics are put into to reduce the number of overall inserts per query
CREATE TABLE metrics_temp (
    Id INTEGER PRIMARY KEY,
    Address TEXT DEFAULT '',
    ClientName TEXT DEFAULT '',
    QueryDomain TEXT DEFAULT '',
    QueryType TEXT DEFAULT '',
    ListName TEXT DEFAULT '',
    ListShortName TEXT DEFAULT '',
    Rule TEXT DEFAULT ''
);