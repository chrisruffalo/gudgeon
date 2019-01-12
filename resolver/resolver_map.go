package resolver

import (
	"github.com/miekg/dns"

	"github.com/chrisruffalo/gudgeon/cache"
	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/util"
)

// a group of resolvers
type resolverMap struct {
	cache     cache.Cache
	resolvers map[string]Resolver
}

type ResolverMap interface {
	Answer(resolverName string, request *dns.Msg) (*dns.Msg, *ResolutionResult, error)
	AnswerMultiResolvers(resolverNames []string, request *dns.Msg) (*dns.Msg, *ResolutionResult, error)
	answerWithContext(resolverName string, context *ResolutionContext, request *dns.Msg) (*dns.Msg, *ResolutionResult, error)
	Cache() cache.Cache
}

// returned as part of resolution to get data what actually resolved the query
type ResolutionResult struct {
	Cached   bool
	Source   string
	Resolver string
}

func result(context *ResolutionContext) *ResolutionResult {
	if context == nil {
		return nil
	}
	result := new(ResolutionResult)
	result.Cached = context.Cached
	result.Source = context.SourceUsed
	result.Resolver = context.ResolverUsed
	return result
}

// returns a map of resolvers with name->resolver mapping
func NewResolverMap(configuredResolvers []*config.GudgeonResolver) ResolverMap {

	// make a new map resolver
	resolverMap := new(resolverMap)

	// empty map of resolvers
	resolverMap.resolvers = make(map[string]Resolver, 0)
	resolverMap.cache = cache.New()

	// build resolvesrs from configuration
	for _, resolverConfig := range configuredResolvers {
		resolver := newResolver(resolverConfig)
		if resolver != nil {
			resolverMap.resolvers[resolver.name] = resolver
		}
	}

	return resolverMap
}

// base answer function for full resolver map
func (resolverMap *resolverMap) answerWithContext(resolverName string, context *ResolutionContext, request *dns.Msg) (*dns.Msg, *ResolutionResult, error) {
	// if no named resolver in map, return
	resolver, ok := resolverMap.resolvers[resolverName]
	if !ok {
		return nil, nil, nil
	}

	// create context
	if context == nil {
		context = DefaultResolutionContextWithMap(resolverMap)
	} else if util.StringIn(resolverName, context.Visited) { // but if context shows already visisted outright skip the resolver
		return nil, nil, nil
	}

	// get answer
	response, err := resolver.Answer(context, request)
	if err != nil {
		return nil, nil, err
	}

	// return with nil error
	return response, result(context), nil
}

// base answer function for full resolver map
func (resolverMap *resolverMap) Answer(resolverName string, request *dns.Msg) (*dns.Msg, *ResolutionResult, error) {
	// return answer with context
	return resolverMap.answerWithContext(resolverName, nil, request)
}

// answer resolvers in order
func (resolverMap *resolverMap) AnswerMultiResolvers(resolverNames []string, request *dns.Msg) (*dns.Msg, *ResolutionResult, error) {
	context := DefaultResolutionContextWithMap(resolverMap)

	for _, resolverName := range resolverNames {
		response, result, err := resolverMap.answerWithContext(resolverName, context, request)
		if err != nil {
			// todo: log error
			continue
		}
		if !util.IsEmptyResponse(response) {
			// then return
			return response, result, nil
		}
	}

	return nil, nil, nil
}

func (resolverMap *resolverMap) Cache() cache.Cache {
	return resolverMap.cache
}
