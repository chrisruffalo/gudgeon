package resolver

import (
	"github.com/miekg/dns"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/resolver/source"
)

// TODO: implement resolution context so that the visited resolvers can go all the way down
// and we can implement source -> resolver in the source package alone

// a single resolver
type resolver struct {
	name    string
	sources []source.Source
}

// a group of resolvers
type mapResolver struct {
	resolvers map[string]Resolver
}

// a source that uses a resolver as a source
type resolverSource struct {
	name string
	resolverMap *mapResolver
}

type ResolverMap interface {
	Answer(resolverName string, request *dns.Msg) (*dns.Msg, error)
}

type Resolver interface {
	Answer(request *dns.Msg) (*dns.Msg, error)
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
	resolver.sources = make([]source.Source, 0)

	// add sources
	for _, configuredSource := range configuredResolver.Sources {
		source := source.NewSource(configuredSource)
		if source != nil {
			resolver.sources = append(resolver.sources, source)
		}
	}

	return resolver
}

// returns a map of resolvers with name->resolver mapping
func NewResolver(configuredResolvers []*config.GudgeonResolver) ResolverMap {

	// make a new map resolver
	resolverMap := new(mapResolver)

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

// answer a question descending into other resolvers as "sources"
func (resolver *resolver) answerWithResolverSources(mapResolver *mapResolver, visitedResolvers *[]string, request *dns.Msg) (*dns.Msg, error) {
	for _, source := range resolver.sources {
		response, err := source.Answer(request)
		// if error is not nil, come up with a response
		if err != nil {
			continue
		}
		// inspect answer
		if response != nil {
			return response, nil
		}
	}

	return nil, nil
}

// base answer function
func (resolver *resolver) Answer(request *dns.Msg) (*dns.Msg, error) {
	visited := make([]string, 0)
	return resolver.answerWithResolverSources(nil, &visited, request)
}

// base answer function for full resolver map
func (resolverMap *mapResolver) Answer(resolverName string, request *dns.Msg) (*dns.Msg, error) {
	visited := make([]string, 0)
	return resolverMap.answerWithDescent(&visited, resolverName, request)
}

// answer with recursive descent into other resolvers
func (resolverMap *mapResolver) answerWithDescent(visitedResolvers *[]string, resolverName string, request *dns.Msg) (*dns.Msg, error) {




	return nil, nil
}


func (resolverSource *resolverSource) Answer(request *dns.Msg) (*dns.Msg, error) {

	return nil, nil
}