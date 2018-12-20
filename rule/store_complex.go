package rule

type complexStore struct {
	backingStore RuleStore
	complexRules map[uint8]map[string][]Rule
}

func (store *complexStore) Load(group string, rules []Rule) uint64 {
	// make decisions based on the rule type
	counter := uint64(0)
	for idx, rule := range rules {
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

			// rule is nilled out from list forwarded to next component
			rules[idx] = nil

			// add to rule counter for return
			counter++
		}
	}

	// backing store load is handled
	if store.backingStore != nil {
		return counter + store.backingStore.Load(group, rules)
	}

	return counter
}

func (store *complexStore) IsMatchAny(groups []string, domain string) Match {

	// go through the order of application (ALLOW, BLOCK)
	for _, element := range ruleApplyOrder {
		for _, group := range groups {
			// get rules that were stored for that type and group
			complexRules := store.complexRules[element][group]
			// do complex rules that are found
			if complexRules != nil {
				// for each of the rules
				for _, rule := range complexRules {
					// check the rule using the rule logic
					if rule.IsMatch(domain) {
						// whitelist immediately returns
						if element == ALLOW {
							return MatchAllow
						} else {
							return MatchBlock
						}
					}
				}
			}
		}
	}

	// if nothing happened then we need to see what the backing store has to say
	if store.backingStore != nil {
		return store.backingStore.IsMatchAny(groups, domain)
	}

	// otherwise (if no backing store is configured) return no match
	return MatchNone
}

// default implementation of IsMatch
func (store *complexStore) IsMatch(group string, domain string) Match {
	return store.IsMatchAny([]string{group}, domain)
}
