-- indexes for table
DROP INDEX metrics_FromTime_Index;
DROP INDEX metrics_AtTime_Index;

-- drop table to migrate down
DROP TABLE metrics;