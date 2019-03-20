-- drop all schema elements created during migration up
DROP INDEX idx_client_metrics_Address;
DROP TABLE client_metrics;

DROP INDEX idx_domain_metrics_DomainName;
DROP TABLE domain_metrics;

DROP INDEX idx_query_types_QueryType;
DROP TABLE query_types;

DROP INDEX idx_lists_ShortName;
DROP TABLE lists;

DROP INDEX idx_rule_hits_ListId;
DROP INDEX idx_rule_hits_Rule;
DROP INDEX idx_rule_hits_ListIdRule;
DROP TABLE rule_hits;