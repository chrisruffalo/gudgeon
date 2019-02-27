-- table structure
CREATE TABLE metrics (
    FromTime         DATETIME,
    AtTime           DATETIME,
    MetricsJson      TEXT,
    IntervalSeconds  INT
);
-- indexes for table
CREATE INDEX metrics_FromTime_Index ON metrics (FromTime);
CREATE INDEX metrics_AtTime_Index ON metrics (AtTime);
