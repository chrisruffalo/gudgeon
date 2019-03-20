package resolver

import (
	"fmt"
	"strings"

	"github.com/miekg/dns"

	"github.com/chrisruffalo/gudgeon/cache"
	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/rule"
	"github.com/chrisruffalo/gudgeon/util"
)

// a group of resolvers
type resolverMap struct {
	cache     cache.Cache
	resolvers map[string]Resolver
}

type ResolverMap interface {
	Answer(rCon *RequestContext, resolverName string, request *dns.Msg) (*dns.Msg, *ResolutionResult, error)
	AnswerMultiResolvers(rCon *RequestContext, resolverNames []string, request *dns.Msg) (*dns.Msg, *ResolutionResult, error)
	answerWithContext(rCon *RequestContext, resolverName string, context *ResolutionContext, request *dns.Msg) (*dns.Msg, *ResolutionResult, error)
	Cache() cache.Cache
}

// returned as part of resolution to get data what actually resolved the query
type ResolutionResult struct {
	Cached   bool
	Consumer string
	Source   string
	Resolver string
	Message  string // errors/panics/context hints

	// reporting on blocks
	Blocked     bool

	// reporting on matches
	Match     rule.Match          // allowed or blocked
	MatchList *config.GudgeonList // name of blocked list
	MatchRule string              // name of actual rule
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
func NewResolverMap(config *config.GudgeonConfig, configuredResolvers []*config.GudgeonResolver) ResolverMap {

	// make a new map resolver
	resolverMap := new(resolverMap)

	// empty map of resolvers
	resolverMap.resolvers = make(map[string]Resolver, 0)
	if *(config.Storage.CacheEnabled) {
		resolverMap.cache = cache.New()
	}

	// resuse sources
	sharedSources := make(map[string]Source)

	// build resolvesrs from configuration
	for _, resolverConfig := range configuredResolvers {
		resolver := newSharedSourceResolver(resolverConfig, sharedSources)
		if resolver != nil {
			resolverMap.resolvers[resolver.name] = resolver
		}
	}

	return resolverMap
}

// base answer function for full resolver map
func (resolverMap *resolverMap) answerWithContext(rCon *RequestContext, resolverName string, context *ResolutionContext, request *dns.Msg) (*dns.Msg, *ResolutionResult, error) {
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
	response, err := resolver.Answer(rCon, context, request)
	if err != nil {
		return nil, nil, err
	}

	// return with nil error
	return response, result(context), nil
}

// base answer function for full resolver map
func (resolverMap *resolverMap) Answer(rCon *RequestContext, resolverName string, request *dns.Msg) (*dns.Msg, *ResolutionResult, error) {
	// return answer with context
	return resolverMap.answerWithContext(rCon, resolverName, nil, request)
}

// answer resolvers in order
func (resolverMap *resolverMap) AnswerMultiResolvers(rCon *RequestContext, resolverNames []string, request *dns.Msg) (*dns.Msg, *ResolutionResult, error) {
	context := DefaultResolutionContextWithMap(resolverMap)
	if rCon == nil {
		rCon = DefaultRequestContext()
	}

	errors := make([]string, 0)

	for _, resolverName := range resolverNames {
		response, result, err := resolverMap.answerWithContext(rCon, resolverName, context, request)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s", err))
			continue
		}
		if !util.IsEmptyResponse(response) {
			// then return
			return response, result, nil
		}
	}

	if len(errors) > 0 {
		return nil, nil, fmt.Errorf("%s", strings.Join(errors, ","))
	}

	return nil, nil, nil
}

func (resolverMap *resolverMap) Cache() cache.Cache {
	return resolverMap.cache
}
