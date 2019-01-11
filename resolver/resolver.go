package resolver

import (
	"strings"

	"github.com/miekg/dns"
	"github.com/ryanuber/go-glob"

	"github.com/chrisruffalo/gudgeon/config"
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
	search  []string
	sources []Source
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
	resolver.search = configuredResolver.Search
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

// base answer function
func (resolver *resolver) answer(context *ResolutionContext, request *dns.Msg) (*dns.Msg, error) {
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

	response, err := resolver.answer(context, request)
	if err != nil {
		return nil, err
	}

	// if there are available search domains use them
	if (response == nil || len(response.Question) < 1) && len(resolver.search) > 0 {
		for _, sDomain := range resolver.search {
			// skip empty search domains
			if "" == sDomain {
				continue
			}

			// create new question
			searchRequest := request.Copy()

			// determine search domain
			searchDomain := searchRequest.Question[0].Name
			if !strings.HasSuffix(searchDomain, ".") {
				searchDomain = searchDomain + "."
			}
			searchDomain = searchDomain + sDomain
			if !strings.HasSuffix(searchDomain, ".") {
				searchDomain = searchDomain + "."
			}

			// update question
			searchRequest.Question[0].Name = searchDomain

			// ask new question
			searchResponse, err := resolver.answer(context, searchRequest)
			if err != nil {
				// todo: log
				continue
			}

			// if the search response is not nil, return it
			if searchResponse != nil {
				// but first update answers to original question domain
				for _, answer := range searchResponse.Answer {
					answer.Header().Name = request.Question[0].Name
				}
				searchResponse.SetReply(request)
				// then return
				return searchResponse, nil
			}
		}
	}

	return response, nil
}
