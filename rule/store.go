 package rule

type RuleStore interface {
	Get(group string) Rule
	Store(group string, rule Rule)
}