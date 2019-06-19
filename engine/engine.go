package engine

import (
	"bytes"
	"database/sql"
	"fmt"
	"net"
	"path"
	"strings"
	"time"

	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/resolver"
	"github.com/chrisruffalo/gudgeon/rule"
	"github.com/chrisruffalo/gudgeon/util"
)

// incomplete list of not-implemented queries
var notImplemented = map[uint16]bool{
	dns.TypeNone: true,
	dns.TypeNULL: true,
	dns.TypeIXFR: true,
	dns.TypeAXFR: true,
}

// an active group is a group within the engine
// that has been processed and is being used to
// select rules. this will be used with the
// rule processing to create rules and will
// be used by the consumer to talk to the store
type group struct {
	engine *engine

	configGroup *config.GudgeonGroup

	lists []*config.GudgeonList
}

// represents a parsed "consumer" type that
// links it to active parsed groups
type consumer struct {
	// engine pointer so we can use the engine from the active consumer
	engine *engine

	// configuration that this consumer was parsed from
	configConsumer *config.GudgeonConsumer

	// list of parsed groups that belong to this consumer
	groupNames []string

	// list of parsed resolvers that belong to this consumer
	resolverNames []string

	// applicable lists
	lists []*config.GudgeonList
}

// stores the internals of the engine abstraction
type engine struct {
	// the session (which will represent the on-disk location inside of the gudgeon folder)
	// that is being used as backing storage and state behind the engine
	session string

	// database for long term data storage
	db *sql.DB

	// metrics instance for engine
	metrics Metrics

	// qlog instance for engine
	qlog QueryLog

	// recorder - combined query data recorder
	recorder *recorder

	// maintain config pointer
	config *config.GudgeonConfig

	// consumers that have been parsed
	consumers []*consumer
	// map for out-of-order consumer getting
	consumerMap map[string]*consumer

	// default consumer
	defaultConsumer *consumer

	// the default group (used to ensure we have one)
	defaultGroup *group

	// the backing store for block/allow rules
	store rule.RuleStore

	// the resolution structure
	resolvers resolver.ResolverMap

	// map of group names to processed/configured engine groups
	groups map[string]*group
}

func (engine *engine) Root() string {
	return path.Join(engine.config.SessionRoot(), engine.session)
}

func (engine *engine) ListPath(listType string) string {
	return path.Join(engine.Root(), listType+".list")
}

type Engine interface {
	IsDomainRuleMatched(consumer *net.IP, domain string) (rule.Match, *config.GudgeonList, string)
	Resolve(domainName string) (string, error)
	Reverse(address string) string

	// different direct handle methods
	Handle(address *net.IP, protocol string, request *dns.Msg) (*dns.Msg, *resolver.RequestContext, *resolver.ResolutionResult)
	HandleWithConsumerName(consumerName string, rCon *resolver.RequestContext, request *dns.Msg) (*dns.Msg, *resolver.RequestContext, *resolver.ResolutionResult)
	HandleWithConsumer(consumer *consumer, rCon *resolver.RequestContext, request *dns.Msg) (*dns.Msg, *resolver.RequestContext, *resolver.ResolutionResult)
	HandleWithGroups(groups []string, rCon *resolver.RequestContext, request *dns.Msg) (*dns.Msg, *resolver.RequestContext, *resolver.ResolutionResult)
	HandleWithResolvers(resolvers []string, rCon *resolver.RequestContext, request *dns.Msg) (*dns.Msg, *resolver.RequestContext, *resolver.ResolutionResult)

	// stats
	CacheSize() int64

	// inner providers
	QueryLog() QueryLog
	Metrics() Metrics

	// close engine and all resources
	Close()

	// shutdown engine and all threads
	Shutdown()
}

func (engine *engine) getConsumerForIP(consumerIP *net.IP) *consumer {
	var foundConsumer *consumer

	for _, activeConsumer := range engine.consumers {
		for _, match := range activeConsumer.configConsumer.Matches {
			// test ip match
			if "" != match.IP {
				matchIP := net.ParseIP(match.IP)
				if matchIP != nil && bytes.Compare(matchIP.To16(), consumerIP.To16()) == 0 {
					foundConsumer = activeConsumer
					break
				}
			}
			// test range match
			if foundConsumer == nil && match.Range != nil && "" != match.Range.Start && "" != match.Range.End {
				startIP := net.ParseIP(match.Range.Start)
				endIP := net.ParseIP(match.Range.End)
				if startIP != nil && endIP != nil && bytes.Compare(consumerIP.To16(), startIP.To16()) >= 0 && bytes.Compare(consumerIP.To16(), endIP.To16()) <= 0 {
					foundConsumer = activeConsumer
					break
				}
			}
			// test net (subnet) match
			if foundConsumer == nil && "" != match.Net {
				_, parsedNet, err := net.ParseCIDR(match.Net)
				if err == nil && parsedNet != nil && parsedNet.Contains(*consumerIP) {
					foundConsumer = activeConsumer
					break
				}
			}
			if foundConsumer != nil {
				break
			}
		}
		if foundConsumer != nil {
			break
		}
	}

	// return default consumer
	if foundConsumer == nil {
		foundConsumer = engine.defaultConsumer
	}

	return foundConsumer
}

func (engine *engine) getConsumerGroups(consumerIP *net.IP) []string {
	consumer := engine.getConsumerForIP(consumerIP)
	return engine.getGroups(consumer)
}

func (engine *engine) getGroups(consumer *consumer) []string {
	// return found consumer data if something was found
	if consumer != nil && len(consumer.groupNames) > 0 {
		return consumer.groupNames
	}

	// return the default group in the event nothing else is available
	return []string{"default"}
}

func (engine *engine) getConsumerResolvers(consumerIP *net.IP) []string {
	consumer := engine.getConsumerForIP(consumerIP)
	return engine.getResolvers(consumer)
}

func (engine *engine) getResolvers(consumer *consumer) []string {
	// return found consumer data if something was found
	if consumer != nil && len(consumer.resolverNames) > 0 {
		return consumer.resolverNames
	}

	// return the default resolver in the event nothing else is available
	return []string{"default"}
}

// return if the domain matches any rule
func (engine *engine) IsDomainRuleMatched(consumerIp *net.IP, domain string) (rule.Match, *config.GudgeonList, string) {
	// get consumer
	consumer := engine.getConsumerForIP(consumerIp)
	return engine.domainRuleMatchedForConsumer(consumer, domain)
}

func (engine *engine) domainRuleMatchForLists(lists []*config.GudgeonList, domain string) (rule.Match, *config.GudgeonList, string) {
	// drop ending . if present from domain
	if strings.HasSuffix(domain, ".") {
		domain = domain[:len(domain)-1]
	}

	// sometimes (in testing, downloading) the store mechanism is nil/unloaded
	if engine.store == nil {
		return rule.MatchNone, nil, ""
	}

	// if no lists are provided, no match
	if len(lists) < 1 {
		return rule.MatchNone, nil, ""
	}

	// return match values
	return engine.store.FindMatch(lists, domain)
}

func (engine *engine) domainRuleMatchedForConsumer(consumer *consumer, domain string) (rule.Match, *config.GudgeonList, string) {
	if consumer == nil {
		return rule.MatchNone, nil, ""
	}
	return engine.domainRuleMatchForLists(consumer.lists, domain)
}

func (engine *engine) domainRuleMatchedForGroups(groups []string, domain string) (rule.Match, *config.GudgeonList, string) {
	if len(groups) < 1 {
		return rule.MatchNone, nil, ""
	}

	// select all lists from found groups
	lists := make([]*config.GudgeonList, 0)
	for _, g := range groups {
		if group, found := engine.groups[g]; found {
			lists = append(lists, group.lists...)
		}
	}

	return engine.domainRuleMatchForLists(lists, domain)
}

// handles recursive resolution of cnames
func (engine *engine) handleCnameResolution(resolvers []string, rCon *resolver.RequestContext, originalRequest *dns.Msg, originalResponse *dns.Msg) *dns.Msg {
	// scope provided finding response
	var response *dns.Msg

	// guard
	if originalResponse == nil || len(originalResponse.Answer) < 1 || originalRequest == nil || len(originalRequest.Question) < 1 {
		return nil
	}

	// if the (first) response is a CNAME then repeat the question but with the cname instead
	if originalResponse.Answer[0] != nil && originalResponse.Answer[0].Header() != nil && originalResponse.Answer[0].Header().Rrtype == dns.TypeCNAME && originalRequest.Question[0].Qtype != dns.TypeCNAME {
		cnameRequest := originalRequest.Copy()
		answer := originalResponse.Answer[0]
		newName := answer.(*dns.CNAME).Target
		cnameRequest.Question[0].Name = newName

		var cnameResponse *dns.Msg
		if len(rCon.Groups) > 0 {
			cnameResponse, _, _ = engine.HandleWithGroups(rCon.Groups, rCon, cnameRequest)
		} else {
			cnameResponse, _, _ = engine.HandleWithResolvers(resolvers, rCon, cnameRequest)
		}
		if cnameResponse != nil && !util.IsEmptyResponse(cnameResponse) {
			// use response
			response = cnameResponse
			// update answer name
			for _, answer := range response.Answer {
				answer.Header().Name = originalRequest.Question[0].Name
			}
			// but set reply as original request
			response.SetReply(originalRequest)
		}
	}

	return response
}

func (engine *engine) invalidRequestHandler(request *dns.Msg) (bool, *dns.Msg) {
	// scope provided finding response
	var (
		response *dns.Msg
	)

	// drop questions that don't meet minimum requirements
	if request == nil || len(request.Question) < 1 {
		response = new(dns.Msg)
		response.SetReply(request)
		response.Rcode = dns.RcodeRefused
		return false, response
	}

	// drop questions that aren't implemented
	qType := request.Question[0].Qtype
	if _, found := notImplemented[qType]; found {
		response = new(dns.Msg)
		response.SetReply(request)
		response.Rcode = dns.RcodeNotImplemented
		return false, response
	}

	// get domain name
	domain := request.Question[0].Name

	// drop questions for domain names that could be malicious/malformed
	if len(domain) < 1 || len(domain) > 255 {
		response = new(dns.Msg)
		response.SetReply(request)
		response.Rcode = dns.RcodeBadName
		return false, response
	}

	return true, nil
}

func (engine *engine) HandleWithResolvers(resolverNames []string, rCon *resolver.RequestContext, request *dns.Msg) (*dns.Msg, *resolver.RequestContext, *resolver.ResolutionResult) {
	// scope provided finding response
	var (
		response *dns.Msg
		err      error
	)

	// create new result
	result := &resolver.ResolutionResult{}

	// handle a request that might not be valid by returning it immediately
	if valid, response := engine.invalidRequestHandler(request); !valid {
		return response, rCon, result
	}

	// we are only doing resolution if there are resolvers to resolve against, otherwise
	// we can skip this part and just return an NXDOMAIN
	if len(resolverNames) > 0 {
		// resolve the question using all of the resolvers specified
		response, result, err = engine.resolvers.AnswerMultiResolvers(rCon, resolverNames, request)
		if err != nil {
			log.Errorf("Could not resolve <%s>: %s", request.Question[0].Name, err)
		} else {
			cnameResponse := engine.handleCnameResolution(resolverNames, rCon, request, response)
			if !util.IsEmptyResponse(cnameResponse) {
				response = cnameResponse
			}
		}
	}

	// if no response is found at this point ensure it is created
	if response == nil {
		response = new(dns.Msg)
	}

	// set codes for response/reply
	if util.IsEmptyResponse(response) {
		response.SetReply(request)
		response.Rcode = dns.RcodeNameError
	}

	// recover during response... this isn't the best golang paradigm but if we don't
	// do this then dns just stops and the entire executable crashes and we stop getting
	// resolution. if you're eating your own dogfood on this one then you lose DNS until
	// you can find and fix the bug which is not ideal.
	if recovery := recover(); recovery != nil {
		response = new(dns.Msg)
		response.SetReply(request)
		response.Rcode = dns.RcodeServerFailure

		// add panic reason to result
		result.Message = fmt.Sprintf("%v", recovery)
	}

	// new result
	if result == nil {
		result = &resolver.ResolutionResult{}
	}

	// return result
	return response, rCon, result
}

func (engine *engine) HandleWithGroups(groups []string, rCon *resolver.RequestContext, request *dns.Msg) (*dns.Msg, *resolver.RequestContext, *resolver.ResolutionResult) {
	// create new result
	result := &resolver.ResolutionResult{}

	// handle a request that might not be valid by returning it immediately
	if valid, response := engine.invalidRequestHandler(request); !valid {
		return response, rCon, result
	}

	// on a valid request add group names to context
	if rCon == nil {
		rCon = &resolver.RequestContext{}
	}
	rCon.Groups = groups

	match, list, ruleText := engine.domainRuleMatchedForGroups(groups, request.Question[0].Name)
	if match != rule.MatchNone {
		result.Match = match
		result.MatchList = list
		result.MatchRule = ruleText
	}

	// handle blocking at the group level
	if match == rule.MatchBlock {
		// todo: do configured action here
		response := new(dns.Msg)
		response.SetReply(request)
		response.Rcode = dns.RcodeNameError
		return response, rCon, result
	}

	// accumulate resolver names
	resolverNames := make([]string, 0)

	// get the resolver names for the groups, in the given order
	for _, groupName := range groups {
		// get resolvers from group
		group, found := engine.groups[groupName]
		if !found {
			continue
		}

		resolverNames = append(resolverNames, group.configGroup.Resolvers...)
	}

	return engine.HandleWithResolvers(resolverNames, rCon, request)
}

func (engine *engine) HandleWithConsumerName(consumerName string, rCon *resolver.RequestContext, request *dns.Msg) (*dns.Msg, *resolver.RequestContext, *resolver.ResolutionResult) {
	consumer, found := engine.consumerMap[consumerName]
	if !found {
		consumer = engine.defaultConsumer
	}
	return engine.HandleWithConsumer(consumer, rCon, request)
}

func (engine *engine) HandleWithConsumer(consumer *consumer, rCon *resolver.RequestContext, request *dns.Msg) (*dns.Msg, *resolver.RequestContext, *resolver.ResolutionResult) {
	// get consumer block status and refuse the request
	if consumer.configConsumer.Block {
		result := &resolver.ResolutionResult{
			Consumer: consumer.configConsumer.Name,
			Blocked:  true,
		}

		response := new(dns.Msg)
		response.Rcode = dns.RcodeRefused
		response.SetReply(request)
		return response, rCon, result
	}

	// get groups for consumer
	groups := engine.getGroups(consumer)

	// return group response
	response, rCon, result := engine.HandleWithGroups(groups, rCon, request)

	// update/set
	if consumer != nil && consumer.configConsumer != nil {
		result.Consumer = consumer.configConsumer.Name
	}

	return response, rCon, result
}

func (engine *engine) Resolve(domainName string) (string, error) {
	m := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Authoritative:     true,
			AuthenticatedData: true,
			RecursionDesired:  true,
			Opcode:            dns.OpcodeQuery,
		},
	}

	if domainName == "" {
		return domainName, fmt.Errorf("cannot resolve an empty domain name")
	}

	// ensure the domain name is fully qualified
	domainName = dns.Fqdn(domainName)

	// make question parts
	m.Question = make([]dns.Question, 1)
	m.Question[0] = dns.Question{Name: domainName, Qtype: dns.TypeA, Qclass: dns.ClassINET}

	// get just response from default consumer
	response, _, _ := engine.HandleWithConsumer(engine.defaultConsumer, &resolver.RequestContext{Protocol: "udp"}, m)

	// return answer
	return util.GetFirstIPResponse(response), nil
}

// return the reverse lookup details for an address and return the result of the (first) ptr record
func (engine *engine) Reverse(address string) string {
	// cannot do reverse lookup
	if address == "" || net.ParseIP(address) == nil {
		return ""
	}

	m := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Authoritative:     true,
			AuthenticatedData: true,
			RecursionDesired:  true,
			Opcode:            dns.OpcodeQuery,
		},
	}

	// we already checked and know this won't be nil
	ip := net.ParseIP(address)

	// make question parts
	m.Question = []dns.Question{{Name: util.ReverseLookupDomain(&ip), Qtype: dns.TypePTR, Qclass: dns.ClassINET}}

	// get just response using all groups
	allGroups := make([]string, 0, len(engine.groups))
	for _, g := range engine.groups {
		allGroups = append(allGroups, g.configGroup.Name)
	}
	response, _, _ := engine.HandleWithGroups(allGroups, &resolver.RequestContext{Protocol: "udp"}, m)

	// look for first pointer
	for _, answer := range response.Answer {
		if aRecord, ok := answer.(*dns.PTR); ok {
			if aRecord != nil && aRecord.Ptr != "" {
				return strings.TrimSpace(aRecord.Ptr)
			}
		}
	}

	// return empty answer if none found answer
	return ""
}

// entry point for external handler
func (engine *engine) Handle(address *net.IP, protocol string, request *dns.Msg) (*dns.Msg, *resolver.RequestContext, *resolver.ResolutionResult) {
	// get consumer
	consumer := engine.getConsumerForIP(address)

	// create context
	rCon := resolver.DefaultRequestContext()
	rCon.Protocol = protocol

	// get results
	response, rCon, result := engine.HandleWithConsumer(consumer, rCon, request)
	finishedTime := time.Now()

	// log them if recorder is active
	if engine.recorder != nil {
		engine.recorder.queue(address, request, response, rCon, result, &finishedTime)
	}

	// return only the result
	return response, rCon, result
}

func (engine *engine) CacheSize() int64 {
	if engine.resolvers != nil && engine.resolvers.Cache() != nil {
		return int64(engine.resolvers.Cache().Size())
	}
	return 0
}

func (engine *engine) Metrics() Metrics {
	return engine.metrics
}

func (engine *engine) QueryLog() QueryLog {
	return engine.qlog
}

// clear lists and remove references
func (engine *engine) Close() {
	// close sources
	engine.resolvers.Close()
	// close rule store
	engine.store.Close()
	// clear references
	engine.db = nil
	engine.qlog = nil
	engine.metrics = nil
	engine.qlog = nil
}

func (engine *engine) Shutdown() {
	// shutting down the recorder shuts down
	// other elements in turn
	if nil != engine.recorder {
		engine.recorder.shutdown()
	}

	// close db
	if nil != engine.db {
		err := engine.db.Close()
		if err != nil {
			log.Errorf("Closing database: %s", err)
		}
	}

	// finish by closing engine
	engine.Close()
}
