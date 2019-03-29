-- move old client metrics table
ALTER TABLE client_metrics RENAME TO _client_metrics_old;
DROP INDEX idx_client_metrics_Address;

-- create new table for storing client names
CREATE TABLE client_names (
    Address TEXT PRIMARY KEY DEFAULT '',
    ClientName TEXT DEFAULT ''
) WITHOUT ROWID;
CREATE INDEX idx_client_names_Address ON client_names (Address);

-- create new table for storing client metrics, removing the ClientName field again
CREATE TABLE client_metrics (
    Address TEXT PRIMARY KEY DEFAULT '',
    Count INT
) WITHOUT ROWID;
CREATE INDEX idx_client_metrics_Address ON client_metrics (Address);

-- move values without the client names
INSERT INTO client_metrics (Address, Count) SELECT Address, Count FROM _client_metrics_old WHERE true;
INSERT INTO client_names (Address, ClientName) SELECT Address, ClientName FROM _client_metrics_old WHERE true;

-- drop old metrics table
DROP TABLE _client_metrics_old;