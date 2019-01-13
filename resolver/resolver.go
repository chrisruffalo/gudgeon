package resolver

import (
	"strings"

	"github.com/miekg/dns"
	"github.com/ryanuber/go-glob"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/util"
)

type RequestContext struct {
	Protocol string // the protocol that the request came in with 
}

func DefaultRequestContext() *RequestContext {
	reqCon := new(RequestContext)
	reqCon.Protocol = "udp" // default to udp
	return reqCon
}


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
	skip    []string
	search  []string
	sources []Source
}

type Resolver interface {
	Answer(rCon *RequestContext, context *ResolutionContext, request *dns.Msg) (*dns.Msg, error)
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
	resolver.skip = configuredResolver.SkipDomains
	resolver.search = configuredResolver.Search
	resolver.sources = make([]Source, 0)

	// add literal hostfile source first source if hosts is configured
	if len(configuredResolver.Hosts) > 0 {
		hostfileSource := newHostFileFromHostArray(configuredResolver.Hosts)
		if hostfileSource != nil {
			resolver.sources = append(resolver.sources, hostfileSource)
		}
	}

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
func (resolver *resolver) answer(rCon *RequestContext, context *ResolutionContext, request *dns.Msg) (*dns.Msg, error) {
	// step through sources and return result
	for _, source := range resolver.sources {
		response, err := source.Answer(rCon, context, request)

		if err != nil {
			// todo: log error
			continue
		}

		if !util.IsEmptyResponse(response) {
			//  update the used resolver
			if context != nil && "" == context.ResolverUsed {
				context.ResolverUsed = resolver.name
			}

			return response, nil
		}
	}

	// todo, maybe return more appropriate error?
	return nil, nil
}

func (resolver *resolver) searchDomains(rCon *RequestContext, context *ResolutionContext, request *dns.Msg) (*dns.Msg, error) {
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
		searchResponse, err := resolver.answer(rCon, context, searchRequest)
		if err != nil {
			// todo: log
			continue
		}

		// if the search response is not nil, return it
		if !util.IsEmptyResponse(searchResponse) {
			// but first update answers to original question domain
			for _, answer := range searchResponse.Answer {
				answer.Header().Name = request.Question[0].Name
			}
			// update reply
			searchResponse.SetReply(request)
			// pass along response
			return searchResponse, nil
		}
	}

	return nil, nil
}

// does the domain match
func domainMatches(questionDomain string, domainsToCheck []string) bool {
	// don't answer if the domain doesn't match
	if len(domainsToCheck) > 0 {
		if strings.HasSuffix(questionDomain, ".") {
			questionDomain = questionDomain[:len(questionDomain)-1]
		}

		for _, domain := range domainsToCheck {
			domain = strings.ToLower(domain)
			// domains that contain a * are glob matches
			if strings.Contains(domain, "*") && glob.Glob(domain, questionDomain) {
				return true
			} else if domain == questionDomain || strings.HasSuffix(questionDomain, "."+domain) {
				return true
			}
		}

		return false
	}

	return true
}

func (resolver *resolver) Answer(rCon *RequestContext, context *ResolutionContext, request *dns.Msg) (*dns.Msg, error) {
	// guard against invalid requests or requests that don't ask anything
	if request == nil || len(request.Question) < 1 {
		return nil, nil
	}

	// get the question name so we can use it to determine if the domain matches
	qname := strings.ToLower(request.Question[0].Name)
	if (len(resolver.domains) > 0 && !domainMatches(qname, resolver.domains)) || (len(resolver.skip) > 0 && domainMatches(qname, resolver.skip)) {
		return nil, nil
	}

	// create context if context is nil (no map)
	if rCon == nil {
		rCon = DefaultRequestContext()
	}
	if context == nil {
		context = DefaultResolutionContext()
	}

	// mark resolver as visited by adding the resolver name to the list of visited resolvers
	context.Visited = append(context.Visited, resolver.name)

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

	response, err := resolver.answer(rCon, context, request)
	if err != nil {
		return nil, err
	}

	// if there are available search domains use them
	if util.IsEmptyResponse(response) && len(resolver.search) > 0 {
		r, err := resolver.searchDomains(rCon, context, request)
		if err == nil && !util.IsEmptyResponse(r) {
			response = r
		}
	}

	// only cache non-nil response
	if context != nil && context.ResolverMap != nil && !context.Stored && response != nil && !response.MsgHdr.Truncated {
		// set as stored based on status of cache action
		context.Stored = context.ResolverMap.Cache().Store(resolver.name, request, response)
	}

	return response, nil
}
