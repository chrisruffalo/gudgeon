package engine

import (
	"bytes"
	"fmt"
	"net"
	"path"
	"strings"

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

	// maintain config pointer
	config *config.GudgeonConfig

	// consumers that have been parsed
	consumers []*consumer

	// default consumer
	defaultConsumer *consumer

	// the default group (used to ensure we have one)
	defaultGroup *group

	// the backing store for block/allow rules
	store rule.RuleStore

	// the resolution structure
	resolvers resolver.ResolverMap
}

func (engine *engine) Root() string {
	return path.Join(engine.config.SessionRoot(), engine.session)
}

func (engine *engine) ListPath(listType string) string {
	return path.Join(engine.Root(), listType+".list")
}

type Engine interface {
	IsDomainBlocked(consumer *net.IP, domain string) (bool, *config.GudgeonList, string)
	Resolve(domainName string) (string, error)
	Reverse(address string) string
	Handle(address *net.IP, protocol string, dnsWriter dns.ResponseWriter, request *dns.Msg) (*dns.Msg, *resolver.RequestContext, *resolver.ResolutionResult)
	CacheSize() int64
}

func (engine *engine) getConsumerForIp(consumerIp *net.IP) *consumer {
	var foundConsumer *consumer

	for _, activeConsumer := range engine.consumers {
		for _, match := range activeConsumer.configConsumer.Matches {
			// test ip match
			if "" != match.IP {
				matchIp := net.ParseIP(match.IP)
				if matchIp != nil && bytes.Compare(matchIp.To16(), consumerIp.To16()) == 0 {
					foundConsumer = activeConsumer
					break
				}
			}
			// test range match
			if foundConsumer == nil && match.Range != nil && "" != match.Range.Start && "" != match.Range.End {
				startIp := net.ParseIP(match.Range.Start)
				endIp := net.ParseIP(match.Range.End)
				if startIp != nil && endIp != nil && bytes.Compare(consumerIp.To16(), startIp.To16()) >= 0 && bytes.Compare(consumerIp.To16(), endIp.To16()) <= 0 {
					foundConsumer = activeConsumer
					break
				}
			}
			// test net (subnet) match
			if foundConsumer == nil && "" != match.Net {
				_, parsedNet, err := net.ParseCIDR(match.Net)
				if err == nil && parsedNet != nil && parsedNet.Contains(*consumerIp) {
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

func (engine *engine) getConsumerGroups(consumerIp *net.IP) []string {
	consumer := engine.getConsumerForIp(consumerIp)
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

func (engine *engine) getConsumerResolvers(consumerIp *net.IP) []string {
	consumer := engine.getConsumerForIp(consumerIp)
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

// return if the domain is blocked, if it is blocked return the list and rule
func (engine *engine) IsDomainBlocked(consumerIp *net.IP, domain string) (bool, *config.GudgeonList, string) {
	// get consumer
	consumer := engine.getConsumerForIp(consumerIp)
	return engine.domainBlockedForConsumer(consumer, domain)
}

func (engine *engine) domainBlockedForConsumer(consumer *consumer, domain string) (bool, *config.GudgeonList, string) {
	// drop ending . if present from domain
	if strings.HasSuffix(domain, ".") {
		domain = domain[:len(domain)-1]
	}

	// sometimes (in testing, downloading) the store mechanism is nil/unloaded
	if engine.store == nil {
		return false, nil, ""
	}

	// look in lists for match
	result, list, ruleText := engine.store.FindMatch(consumer.lists, domain)

	return !(result == rule.MatchAllow || result == rule.MatchNone), list, ruleText
}

// handles recursive resolution of cnames
func (engine *engine) handleCnameResolution(address *net.IP, protocol string, originalRequest *dns.Msg, originalResponse *dns.Msg) *dns.Msg {
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
		cnameResponse, _, _ := engine.performRequest(address, protocol, cnameRequest)
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

func (engine *engine) performRequest(address *net.IP, protocol string, request *dns.Msg) (*dns.Msg, *resolver.RequestContext, *resolver.ResolutionResult) {
	// scope provided finding response
	var (
		response *dns.Msg
		result   *resolver.ResolutionResult
	)

	// create context
	rCon := resolver.DefaultRequestContext()
	rCon.Protocol = protocol

	// drop questions that don't meet minimum requirements
	if request == nil || len(request.Question) < 1 {
		response = new(dns.Msg)
		response.SetReply(request)
		response.Rcode = dns.RcodeRefused
		return response, rCon, result
	}

	// drop questions that aren't implemented
	qType := request.Question[0].Qtype
	if _, found := notImplemented[qType]; found {
		response = new(dns.Msg)
		response.SetReply(request)
		response.Rcode = dns.RcodeNotImplemented
		return response, rCon, result
	}

	// get domain name
	domain := request.Question[0].Name

	// get consumer
	consumer := engine.getConsumerForIp(address)

	// get block status and just hang up if blocked at the consumer level
	if consumer.configConsumer.Block {
		result = new(resolver.ResolutionResult)
		result.Blocked = true
	} else if blocked, list, ruleText := engine.domainBlockedForConsumer(consumer, domain); blocked {
		// set blocked values
		result = new(resolver.ResolutionResult)
		result.Blocked = true
		result.BlockedList = list
		result.BlockedRule = ruleText
	} else {
		// if not blocked then actually try resolution, by grabbing the resolver names
		resolvers := engine.getResolvers(consumer)
		r, res, err := engine.resolvers.AnswerMultiResolvers(rCon, resolvers, request)
		if err != nil {
			log.Errorf("Could not resolve <%s> for consumer '%s': %s", domain, consumer.configConsumer.Name, err)
		} else {
			response = r
			result = res
			cnameResponse := engine.handleCnameResolution(address, protocol, request, response)
			if cnameResponse != nil {
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

	// set consumer
	if result != nil && consumer != nil && consumer.configConsumer != nil {
		result.Consumer = consumer.configConsumer.Name
	}

	// return result
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

	// get just response
	address := net.ParseIP("127.0.0.1")
	response, _, _ := engine.performRequest(&address, "udp", m)

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
	m.Question = make([]dns.Question, 1)
	m.Question[0] = dns.Question{Name: util.ReverseLookupDomain(&ip), Qtype: dns.TypePTR, Qclass: dns.ClassINET}

	// get just response
	client := net.ParseIP("127.0.0.1")
	response, _, _ := engine.performRequest(&client, "udp", m)

	// look for first pointer
	for _, answer := range response.Answer {
		if aRecord, ok := answer.(*dns.PTR); ok {
			if aRecord != nil && aRecord.Ptr != "" {
				return strings.TrimSpace(aRecord.Ptr)
			}
		}
	}

	// return answer
	return ""
}

func (engine *engine) Handle(address *net.IP, protocol string, dnsWriter dns.ResponseWriter, request *dns.Msg) (*dns.Msg, *resolver.RequestContext, *resolver.ResolutionResult) {
	// return results
	return engine.performRequest(address, protocol, request)
}

func (engine *engine) CacheSize() int64 {
	if engine.resolvers != nil && engine.resolvers.Cache() != nil {
		return int64(engine.resolvers.Cache().Size())
	}
	return 0
}