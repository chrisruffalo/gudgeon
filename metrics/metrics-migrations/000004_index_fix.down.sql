-- recreate indexes
CREATE INDEX idx_rule_hits_Rule on rule_hits (Rule);
CREATE INDEX idx_rule_hits_ListIdRule on rule_hits (ListId, Rule);