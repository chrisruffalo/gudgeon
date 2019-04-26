-- move old table
ALTER TABLE client_metrics RENAME TO _old_client_metrics

-- create new table for storing client metrics
CREATE TABLE client_metrics (
    Address TEXT PRIMARY KEY DEFAULT '',
    ClientName TEXT,
    Count INT
);
CREATE INDEX idx_client_metrics_Address ON client_metrics (Address);

-- insert values from old table
INSERT INTO client_metrics (Address, Count) SELECT Address, Count FROM _old_client_metrics WHERE true;

-- drop old table
DROP TABLE _old_client_metrics;