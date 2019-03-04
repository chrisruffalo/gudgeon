-- drop indexes for table
DROP INDEX qlog_AddressIndex;
DROP INDEX qlog_RequestDomainIndex;
DROP INDEX qlog_BlockedIndex;
DROP INDEX qlog_CreatedIndex;

-- reverse by dropping the qlog table
DROP TABLE qlog;