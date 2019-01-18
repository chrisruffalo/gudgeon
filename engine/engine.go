package engine

import (
	"bytes"
	"fmt"
	"net"
	"path"
	"strings"

	"github.com/miekg/dns"

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
	configConsumer *config.GundgeonConsumer

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
	IsDomainBlocked(consumer net.IP, domain string) (bool, *config.GudgeonList, string)
	Resolve(domainName string) (string, error)
	Handle(dnsWriter dns.ResponseWriter, request *dns.Msg) (*net.IP, *dns.Msg, *resolver.RequestContext, *resolver.ResolutionResult)
}

func (engine *engine) getConsumerForIp(consumerIp net.IP) *consumer {
	var foundConsumer *consumer

	for _, activeConsumer := range engine.consumers {
		for _, match := range activeConsumer.configConsumer.Matches {
			// test ip match
			if "" != match.IP {
				matchIp := net.ParseIP(match.IP)
				if matchIp != nil && bytes.Compare(matchIp.To16(), consumerIp.To16()) == 0 {
					foundConsumer = activeConsumer
				}
			}
			// test range match
			if foundConsumer == nil && match.Range != nil && "" != match.Range.Start && "" != match.Range.End {
				startIp := net.ParseIP(match.Range.Start)
				endIp := net.ParseIP(match.Range.End)
				if startIp != nil && endIp != nil && bytes.Compare(consumerIp.To16(), startIp.To16()) >= 0 && bytes.Compare(consumerIp.To16(), endIp.To16()) <= 0 {
					foundConsumer = activeConsumer
				}
			}
			// test net (subnet) match
			if foundConsumer == nil && "" != match.Net {
				_, parsedNet, err := net.ParseCIDR(match.Net)
				if err == nil && parsedNet != nil && parsedNet.Contains(consumerIp) {
					foundConsumer = activeConsumer
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

func (engine *engine) getConsumerGroups(consumerIp net.IP) []string {
	foundConsumer := engine.getConsumerForIp(consumerIp)

	// return found consumer data if something was found
	if foundConsumer != nil && len(foundConsumer.groupNames) > 0 {
		return foundConsumer.groupNames
	}

	// return the default group in the event nothing else is available
	return []string{"default"}
}

func (engine *engine) getConsumerResolvers(consumerIp net.IP) []string {
	foundConsumer := engine.getConsumerForIp(consumerIp)

	// return found consumer data if something was found
	if foundConsumer != nil && len(foundConsumer.resolverNames) > 0 {
		return foundConsumer.resolverNames
	}

	// return the default resolver in the event nothing else is available
	return []string{"default"}
}

// return if the domain is blocked, if it is blocked return the list and rule
func (engine *engine) IsDomainBlocked(consumerIp net.IP, domain string) (bool, *config.GudgeonList, string) {
	// drop ending . if present from domain
	if strings.HasSuffix(domain, ".") {
		domain = domain[:len(domain)-1]
	}

	// get consumer
	consumer := engine.getConsumerForIp(consumerIp)

	// look in lists for match
	result, list, ruleText := engine.store.FindMatch(consumer.lists, domain)

	return !(result == rule.MatchAllow || result == rule.MatchNone), list, ruleText
}

// handles recursive resolution of cnames
func (engine *engine) handleCnameResolution(address net.IP, protocol string, originalRequest *dns.Msg, originalResponse *dns.Msg) *dns.Msg {
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
		if cnameResponse != nil && len(cnameResponse.Answer) > 0 {
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

func (engine *engine) performRequest(address net.IP, protocol string, request *dns.Msg) (*dns.Msg, *resolver.RequestContext, *resolver.ResolutionResult) {
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

	// get block status
	if blocked, list, ruleText := engine.IsDomainBlocked(address, domain); blocked {
		// set blocked values
		result = new(resolver.ResolutionResult)
		result.Blocked = true
		result.BlockedList = list
		result.BlockedRule = ruleText

		// just say that the response code is that the answer wasn't found
		response = new(dns.Msg)
		response.SetReply(request)
		response.Rcode = dns.RcodeNameError
	} else {
		// if not blocked then actually try resolution, by grabbing the resolver names
		resolvers := engine.getConsumerResolvers(address)
		r, res, err := engine.resolvers.AnswerMultiResolvers(rCon, resolvers, request)
		if err != nil {
			// todo: log error in resolution
		} else {
			response = r
			result = res
			cnameResponse := engine.handleCnameResolution(address, protocol, request, response)
			if cnameResponse != nil {
				response = cnameResponse
			}
		}
	}

	// if no response is found at this point return NXDOMAIN
	if util.IsEmptyResponse(response) {
		response = new(dns.Msg)
		response.SetReply(request)
		response.Rcode = dns.RcodeNameError
	}

	// recover and log response... this isn't the best golang paradigm but if we don't
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

	if !strings.HasSuffix(domainName, ".") {
		domainName += "."
	}

	// make question parts
	m.Question = make([]dns.Question, 1)
	m.Question[0] = dns.Question{Name: domainName, Qtype: dns.TypeA, Qclass: dns.ClassINET}

	// get just response
	response, _, _ := engine.performRequest(net.ParseIP("127.0.0.1"), "udp", m)

	// return answer
	return util.GetFirstAResponse(response), nil
}

func (engine *engine) Handle(dnsWriter dns.ResponseWriter, request *dns.Msg) (*net.IP, *dns.Msg, *resolver.RequestContext, *resolver.ResolutionResult) {
	// allow us to look up the consumer IP
	var a net.IP

	// get consumer ip from request
	protocol := ""
	if ip, ok := dnsWriter.RemoteAddr().(*net.UDPAddr); ok {
		a = ip.IP
		protocol = "udp"
	}
	if ip, ok := dnsWriter.RemoteAddr().(*net.TCPAddr); ok {
		a = ip.IP
		protocol = "tcp"
	}

	// perform request and get details
	response, rCon, result := engine.performRequest(a, protocol, request)

	// return results
	return &a, response, rCon, result
}
