-- move old client_metrics table
ALTER TABLE client_metrics RENAME TO _client_metrics_old;
DROP INDEX idx_client_metrics_Address;

-- create new table for storing client metrics, re-adding the client name field
CREATE TABLE client_metrics (
    Address TEXT PRIMARY KEY DEFAULT '',
    ClientName TEXT DEFAULT '',
    Count INT
);
CREATE INDEX idx_client_metrics_Address ON client_metrics (Address);

-- insert values joined from address table
INSERT INTO client_metrics (Address, ClientName, Count)
    SELECT cm.Address, cn.ClientName, cm.Count FROM client_metrics cm
    JOIN client_names cn ON cn.Address = cm.Address;

-- drop old client metrics
DROP TABLE _client_metrics_old;

-- drop client names
DROP TABLE client_names;