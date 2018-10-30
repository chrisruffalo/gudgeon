 package rule

type RuleStore interface {
	Load(group string, rules []Rule)
	IsMatch(group string, domain string) bool
}

type complexStore struct {
	backingStore RuleStore
	complexRules map[uint8]map[string][]Rule
}

// order of applying/creating/using rules
var ruleApplyOrder = []uint8{WHITELIST, BLACKLIST, BLOCKLIST}

func CreateStore(backingStoreType string) RuleStore {

	// first create the complex rule store wrapper
	store := new(complexStore)
	store.complexRules = make(map[uint8]map[string][]Rule)
	for _, element := range ruleApplyOrder {
		store.complexRules[element] = make(map[string][]Rule)
	}	

	// create appropriate backing store
	var backingStore RuleStore
	if "mem" == backingStoreType || "memory" == backingStoreType || "" == backingStoreType {
		backingStore = new(memoryStore)
	}
	
	// set backing store
	store.backingStore = backingStore

	// finalize and return store
	return store
} 

func (store *complexStore) Load(group string, rules []Rule) {
	// need to just forward the simple rules
	simpleRuleList := make([]Rule,0)

	// make decisions based on the rule type
	for _, rule := range rules {
		if rule == nil {
			continue
		}

		// complex rules are locally stored
		if rule.IsComplex() {
			targetType := rule.RuleType()
			targetGroup := store.complexRules[targetType][group]
			if targetGroup == nil {
				targetGroup = make([]Rule, 0)
				store.complexRules[targetType][group] = targetGroup
			}
			store.complexRules[targetType][group] = append(store.complexRules[targetType][group], rule)			
		} else {
			// simple rules are forwarded to the backing store for storage
			simpleRuleList = append(simpleRuleList, rule)
		}
	}

	// backing store load is handled
	if store.backingStore != nil {
		store.backingStore.Load(group, simpleRuleList)
	}
}

func (store *complexStore) IsMatch(group string, domain string) bool {

	// go through the order of application (WHITELIST, BLACKLIST, BLOCKLIST)
	for _, element := range ruleApplyOrder {
		// get rules that were stored for that type and group
		complexRules := store.complexRules[element][group]
		// do complex rules that are found
		if complexRules != nil {
			// for each of the rules
			for _, rule := range complexRules {
				// check the rule using the rule logic
				if rule.IsMatch(domain) {
					// whitelist immediately returns false for a match
					if element == WHITELIST {
						return false
					} else {
						return true
					}
				}
			}
		}
	}

	// if nothing happened then we need to see what the backing store has to say
	if store.backingStore != nil {
		return store.backingStore.IsMatch(group, domain)
	}

	// otherwise (if no backing store is configured) return false
	return false
}