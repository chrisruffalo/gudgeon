package resolver

import (
	"github.com/miekg/dns"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/util"
)

type ResolutionContext struct {
	ResolverMap ResolverMap // pointer to the resolvermap that started resolution, can be nil
	Visited     []string    // list of visited resolver names
}

func DefaultContext() *ResolutionContext {
	context := new(ResolutionContext)
	context.Visited = make([]string, 0)
	return context
}

func DefaultContextWithMap(resolverMap ResolverMap) *ResolutionContext {
	context := DefaultContext()
	context.ResolverMap = resolverMap
	return context
}

// a single resolver
type resolver struct {
	name    string
	sources []Source
}

// a group of resolvers
type resolverMap struct {
	resolvers map[string]Resolver
}

type ResolverMap interface {
	Answer(resolverName string, request *dns.Msg) (*dns.Msg, error)
	AnswerMultiResolvers(resolverNames []string, request *dns.Msg) (*dns.Msg, error)
	answerWithContext(resolverName string, context *ResolutionContext, request *dns.Msg) (*dns.Msg, error)
}

type Resolver interface {
	Answer(context *ResolutionContext, request *dns.Msg) (*dns.Msg, error)
}

// create a new resolver
func newResolver(configuredResolver *config.GudgeonResolver) *resolver {
	// resolvers must have a name
	if "" == configuredResolver.Name {
		return nil
	}

	// make new resolver
	resolver := new(resolver)
	resolver.name = configuredResolver.Name
	resolver.sources = make([]Source, 0)

	// add sources
	for _, configuredSource := range configuredResolver.Sources {
		source := NewSource(configuredSource)
		if source != nil {
			resolver.sources = append(resolver.sources, source)
		}
	}

	return resolver
}

// returns a map of resolvers with name->resolver mapping
func NewResolverMap(configuredResolvers []*config.GudgeonResolver) ResolverMap {

	// make a new map resolver
	resolverMap := new(resolverMap)

	// empty map of resolvers
	resolverMap.resolvers = make(map[string]Resolver, 0)

	// build resolvesrs from configuration
	for _, resolverConfig := range configuredResolvers {
		resolver := newResolver(resolverConfig)
		if resolver != nil {
			resolverMap.resolvers[resolver.name] = resolver
		}
	}

	return resolverMap
}

// base answer function
func (resolver *resolver) Answer(context *ResolutionContext, request *dns.Msg) (*dns.Msg, error) {
	// create context if context is nil (no map)
	if context == nil {
		context = DefaultContext()
	}

	// mark resolver as visited by adding the resolver name to the list of visited resolvers
	context.Visited = append(context.Visited, resolver.name)

	// step through sources and return result
	for _, source := range resolver.sources {
		response, err := source.Answer(context, request)
		if err != nil {
			// todo: log error
			continue
		}
		if response != nil && len(response.Answer) > 0 {
			return response, nil
		}
	}

	// todo, maybe return more appropriate error?
	return nil, nil
}

// base answer function for full resolver map
func (resolverMap *resolverMap) Answer(resolverName string, request *dns.Msg) (*dns.Msg, error) {
	// return answer with context
	return resolverMap.answerWithContext(resolverName, nil, request)
}

// answer resolvers in order
func (resolverMap *resolverMap) AnswerMultiResolvers(resolverNames []string, request *dns.Msg) (*dns.Msg, error) {
	context := DefaultContextWithMap(resolverMap)

	for _, resolverName := range resolverNames {
		response, err := resolverMap.answerWithContext(resolverName, context, request)
		if err != nil {
			// todo: log error
			continue
		}
		if response != nil {
			return response, nil
		}
	}

	return nil, nil
}

// base answer function for full resolver map
func (resolverMap *resolverMap) answerWithContext(resolverName string, context *ResolutionContext, request *dns.Msg) (*dns.Msg, error) {
	// if no named resolver in map, return
	resolver, ok := resolverMap.resolvers[resolverName]
	if !ok {
		return nil, nil
	}

	// create context
	if context == nil {
		context = DefaultContextWithMap(resolverMap)
	} else if util.StringIn(resolverName, context.Visited) { // but if context shows already visisted outright skip the resolver
		return nil, nil
	}

	// return answer
	return resolver.Answer(context, request)
}
