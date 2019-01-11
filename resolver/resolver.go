package resolver

import (
	"fmt"
	"strings"

	"github.com/miekg/dns"
	"github.com/ryanuber/go-glob"

	"github.com/chrisruffalo/gudgeon/cache"
	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/util"
)

type ResolutionContext struct {
	// resolution tools / recursive issues
	ResolverMap ResolverMap // pointer to the resolvermap that started resolution, can be nil
	Visited     []string    // list of visited resolver names
	Stored      bool        // has the result been stored already
	// reporting on actual resolver/source
	ResolverUsed string // the resolver that did the work
	SourceUsed   string // actual source that did the resolution
	Cached       bool   // was the result found by querying the Cache
}

func DefaultResolutionContext() *ResolutionContext {
	context := new(ResolutionContext)
	context.Visited = make([]string, 0)
	context.Stored = false
	context.ResolverUsed = ""
	context.SourceUsed = ""
	context.Cached = false
	return context
}

func DefaultResolutionContextWithMap(resolverMap ResolverMap) *ResolutionContext {
	context := DefaultResolutionContext()
	context.ResolverMap = resolverMap
	return context
}

// a single resolver
type resolver struct {
	name    string
	domains []string
	sources []Source
}

// a group of resolvers
type resolverMap struct {
	cache     cache.Cache
	resolvers map[string]Resolver
}

type ResolverMap interface {
	Answer(resolverName string, request *dns.Msg) (*dns.Msg, error)
	AnswerMultiResolvers(resolverNames []string, request *dns.Msg) (*dns.Msg, error)
	answerWithContext(resolverName string, context *ResolutionContext, request *dns.Msg) (*dns.Msg, error)
	Cache() cache.Cache
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
	resolver.domains = configuredResolver.Domains
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

// base answer function
func (resolver *resolver) Answer(context *ResolutionContext, request *dns.Msg) (*dns.Msg, error) {
	// guard against invalid requests or requests that don't ask anything
	if request == nil || len(request.Question) < 1 {
		return nil, nil
	}

	// create context if context is nil (no map)
	if context == nil {
		context = DefaultResolutionContext()
	}

	// mark resolver as visited by adding the resolver name to the list of visited resolvers
	context.Visited = append(context.Visited, resolver.name)

	// don't answer if the domain doesn't match
	if len(resolver.domains) > 0 {
		// get the question name so we can use it to determine if the domain matches
		qname := request.Question[0].Name
		if strings.HasSuffix(qname, ".") {
			qname = qname[:len(qname)-1]
		}

		// default to "no match"
		domainMatches := false

		for _, domain := range resolver.domains {
			// domains that contain a * are glob matches
			if strings.Contains(domain, "*") && glob.Glob(domain, qname) {
				domainMatches = true
				break
			} else if domain == qname || strings.HasSuffix(qname, "."+domain) {
				// strings that do not are raw domain/subdomain matches
				domainMatches = true
				break
			}
		}

		// the domain doesn't match any of the available domains so
		// we bail out
		if !domainMatches {
			return nil, nil
		}
	}

	// check cache first (if available)
	if context.ResolverMap != nil {
		cachedResponse, found := context.ResolverMap.Cache().Query(resolver.name, request)
		if found && cachedResponse != nil {
			if "" == context.ResolverUsed {
				context.ResolverUsed = resolver.name
			}
			// set as stored in the context because it was found int he cache
			context.Stored = true
			context.Cached = true
			return cachedResponse, nil
		}
	}

	// step through sources and return result
	for _, source := range resolver.sources {
		response, err := source.Answer(context, request)

		if err != nil {
			// todo: log error
			continue
		}

		if response != nil && (len(response.Answer) > 0 || len(response.Ns) > 0 || len(response.Extra) > 0) {
			//  update the used resolver
			if context != nil && "" == context.ResolverUsed {
				context.ResolverUsed = resolver.name
			}

			// only cache non-nil response
			if context != nil && context.ResolverMap != nil && response != nil && !context.Stored {
				// save cached answer
				context.ResolverMap.Cache().Store(resolver.name, request, response)
				// set as stored
				context.Stored = true
			}

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
	context := DefaultResolutionContextWithMap(resolverMap)

	for _, resolverName := range resolverNames {
		response, err := resolverMap.answerWithContext(resolverName, context, request)
		if err != nil {
			// todo: log error
			continue
		}
		if response != nil {
			// log query
			if context.Cached {
				fmt.Printf("[%s] Q << %s >> from cache\n", context.ResolverUsed, request.Question[0].String())
			} else {
				fmt.Printf("[%s] Q << %s >> from source: '%s'\n", context.ResolverUsed, request.Question[0].String(), context.SourceUsed)
			}
			// then return
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
		context = DefaultResolutionContextWithMap(resolverMap)
	} else if util.StringIn(resolverName, context.Visited) { // but if context shows already visisted outright skip the resolver
		return nil, nil
	}

	// get answer
	response, err := resolver.Answer(context, request)
	if err != nil {
		return nil, err
	}

	// return with nil error
	return response, nil
}

func (resolverMap *resolverMap) Cache() cache.Cache {
	return resolverMap.cache
}
