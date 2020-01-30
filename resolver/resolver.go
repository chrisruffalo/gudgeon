package resolver

import (
	"github.com/ryanuber/go-glob"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/util"
)

var requestContextPool = sync.Pool{
	New: func() interface{} {
		return &RequestContext{}
	},
}

// additional information passed along with the request
type RequestContext struct {
	Started  time.Time // when the request starts
	Protocol string    // the protocol that the request came in with
	Groups   []string  // the groups that belong to the original requester

	// pool reference for returning
	pool *sync.Pool
}

func DefaultRequestContext() *RequestContext {
	reqCon := requestContextPool.Get().(*RequestContext)
	reqCon.Started = time.Now()
	reqCon.Protocol = "udp"
	reqCon.pool = &requestContextPool
	return reqCon
}

func (context *RequestContext) Put() {
	// clear values that won't be set
	context.Groups = make([]string, 0)
	// return to pool for reuse
	if context.pool != nil {
		context.pool.Put(context)
	}
}

var resolutionContextPool = sync.Pool{
	New: func() interface{} {
		return &ResolutionContext{}
	},
}

// information relevant to the process of resolution
type ResolutionContext struct {
	// resolution tools / recursive issues
	ResolverMap ResolverMap // pointer to the resolvermap that started resolution, can be nil
	Visited     []string    // list of visited resolver names
	Stored      bool        // has the result been stored already

	// reporting on actual resolver/source
	ResolverUsed string // the resolver that did the work
	SourceUsed   string // actual source that did the resolution
	Cached       bool   // was the result found by querying the Cache

	// reporting on blocks/block status (todo: make Match not block)
	Blocked     bool
	BlockedList *config.GudgeonList // pointer to blocked list
	BlockedRule string              // name of actual rule

	// pool reference for returning
	pool *sync.Pool
}

func DefaultResolutionContext() *ResolutionContext {
	context := resolutionContextPool.Get().(*ResolutionContext)
	context.Visited = make([]string, 0)
	context.Stored = false
	context.ResolverUsed = ""
	context.SourceUsed = ""
	context.Cached = false
	context.Blocked = false
	context.BlockedRule = ""

	context.pool = &resolutionContextPool
	return context
}

func DefaultResolutionContextWithMap(resolverMap ResolverMap) *ResolutionContext {
	context := DefaultResolutionContext()
	context.ResolverMap = resolverMap
	return context
}

func (context *ResolutionContext) Put() {
	if context.pool != nil {
		context.pool.Put(context)
	}
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
	Close()
}

func newResolver(configuredResolver *config.GudgeonResolver) *resolver {
	return newSharedSourceResolver(configuredResolver, nil, nil)
}

// create a new resolver
func newSharedSourceResolver(configuredResolver *config.GudgeonResolver, configuredSources map[string]Source, sharedSources map[string]Source) *resolver {
	// resolvers must have a name
	if "" == configuredResolver.Name {
		return nil
	}

	// fix maps
	if configuredSources == nil {
		configuredSources = make(map[string]Source, 0)
	}
	if sharedSources == nil {
		sharedSources = make(map[string]Source, 0)
	}

	// make new resolver
	resolver := &resolver{
		name:    configuredResolver.Name,
		domains: configuredResolver.Domains,
		skip:    configuredResolver.SkipDomains,
		search:  configuredResolver.Search,
		sources: make([]Source, 0, len(configuredResolver.Sources)),
	}

	// add literal hostfile source first source if hosts is configured
	if len(configuredResolver.Hosts) > 0 {
		hostfileSource := &hostFileSource{}
		hostfileSource.LoadArray(configuredResolver.Hosts)
		resolver.sources = append(resolver.sources, hostfileSource)
	}

	// add sources
	for _, configuredSource := range configuredResolver.Sources {
		// first check if there is a configured source given for the spec,
		// if so, we should use it instead of looking for it
		if cS, found := configuredSources[configuredSource]; found {
			log.Infof("Loaded configured source: %s", cS.Name())
			resolver.sources = append(resolver.sources, cS)
		}

		// special logic for adding the system source to the system resolver
		// otherwise we use the name system and point back to it with a
		// resolver source
		if resolver.name == "system" && configuredSource == "system" {
			resolver.sources = append(resolver.sources, &systemSource{})
		} else {
			var source Source

			if sharedSource, found := sharedSources[configuredSource]; found {
				resolver.sources = append(resolver.sources, sharedSource)
				source = sharedSource
			}

			if source == nil {
				source := NewSource(configuredSource)
				if source != nil && "" != source.Name() {
					log.Infof("Loaded source: %s", source.Name())
					resolver.sources = append(resolver.sources, source)
					sharedSources[configuredSource] = source
				}
			}
		}
	}

	return resolver
}

// base answer function
func (resolver *resolver) answer(rCon *RequestContext, context *ResolutionContext, request *dns.Msg) (*dns.Msg, error) {
	// step through sources and return result
	emptyCounter := 0
	errCounter := 0
	for _, source := range resolver.sources {
		response, err := source.Answer(rCon, context, request)

		if err != nil {
			errCounter++
			log.Errorf("Resolver '%s' for question: '%s': %s", source.Name(), request.Question[0].Name, err)
			continue
		}

		// if the response is not empty and the response is not explicitly NXDOMAIN go on to the next source
		if !util.IsEmptyResponse(response) {
			//  update the used resolver
			if context != nil && "" == context.ResolverUsed {
				context.ResolverUsed = resolver.name
			}

			return response, nil
		}

		// count empty sources
		emptyCounter++
	}

	// log error because no sources managed to resolve in this resolver
	if errCounter > 0 {
		log.Debugf("No response from %d sources (%d empty, %d errors) in resolver: %s", len(resolver.sources), emptyCounter, errCounter, resolver.name)
	}
	return nil, nil
}

func (resolver *resolver) searchDomains(rCon *RequestContext, context *ResolutionContext, request *dns.Msg) (*dns.Msg, error) {
	// create new question
	searchRequest := request.Copy()

	// determine search domain
	originalDomain := searchRequest.Question[0].Name

	// if the original domain is empty, the original domain is just ".", or the original domain has
	// more than one part ("db.local." instead of just "db.") then we aren't going to extend the search
	// domain to cover it
	domainSplits := strings.Split(originalDomain, ".")
	if "" == originalDomain || "." == originalDomain || (len(domainSplits) > 1 && "" != domainSplits[1]) {
		return nil, nil
	}

	for _, sDomain := range resolver.search {
		// skip empty search domains
		if "" == sDomain {
			continue
		}

		// update question in the request to match the searched domain
		searchRequest.Question[0].Name = dns.Fqdn(dns.Fqdn(originalDomain) + sDomain)

		// ask new question
		searchResponse, err := resolver.answer(rCon, context, searchRequest)
		if err != nil {
			log.Errorf("During domain search: %s", err)
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

		for idx := 0; idx < len(domainsToCheck); idx++ {
			domain := strings.ToLower(domainsToCheck[idx])
			// domains that contain a * are glob matches
			if strings.Contains(domain, "*") {
				return glob.Glob(domain, questionDomain)
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
	if context.ResolverMap != nil && context.ResolverMap.Cache() != nil {
		cachedResponse, found := context.ResolverMap.Cache().Query(resolver.name, request)
		if found && cachedResponse != nil && !util.IsEmptyResponse(cachedResponse) {
			// if no resolver has been set then use that resolver name as the source name
			if "" == context.ResolverUsed {
				context.ResolverUsed = resolver.name
			}
			// set as stored in the context because it was found in the cache
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
	if context.ResolverMap != nil && context.ResolverMap.Cache() != nil && !context.Stored && response != nil && !response.MsgHdr.Truncated {
		// set as stored based on status of cache action
		context.Stored = context.ResolverMap.Cache().Store(resolver.name, request, response)
	}

	return response, nil
}

func (resolver *resolver) Close() {
	for _, source := range resolver.sources {
		if source != nil {
			log.Debugf("Closing source: %s", source.Name())
			source.Close()
		}
	}
}
