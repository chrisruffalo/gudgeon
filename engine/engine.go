package engine

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/google/uuid"
	"github.com/miekg/dns"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/downloader"
	"github.com/chrisruffalo/gudgeon/qlog"
	"github.com/chrisruffalo/gudgeon/resolver"
	"github.com/chrisruffalo/gudgeon/rule"
	"github.com/chrisruffalo/gudgeon/util"
)

// an active group is a group within the engine
// that has been processed and is being used to
// select rules. this will be used with the
// rule processing to create rules and will
// be used by the consumer to talk to the store
type group struct {
	engine *engine

	configGroup *config.GudgeonGroup
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
	IsDomainBlocked(consumer net.IP, domain string) bool
	Handle(dnsWriter dns.ResponseWriter, request *dns.Msg)
}

// returns an array of the GudgeonLists that are assigned either by name or by tag from within the list of GudgeonLists in the config file
func assignedLists(listNames []string, listTags []string, lists []*config.GudgeonList) []*config.GudgeonList {
	// empty list
	should := []*config.GudgeonList{}
	// a list with no tags has the "default" tag
	if len(listTags) < 1 {
		listTags = []string{"default"}
	}

	// check names
	for _, list := range lists {
		if util.StringIn(list.Name, listNames) {
			should = append(should, list)
			continue
		}

		// if there are no tags the tag is "default"
		checkingTags := list.Tags
		if len(checkingTags) < 1 {
			checkingTags = []string{"default"}
		}
		for _, tag := range checkingTags {
			if util.StringIn(tag, listTags) {
				should = append(should, list)
				break
			}
		}
	}

	return should
}

func New(conf *config.GudgeonConfig) (Engine, error) {
	// create return object
	engine := new(engine)
	engine.config = conf

	// create store
	engine.store = rule.CreateDefaultStore() // create default store type

	// create session key
	uuid := uuid.New()
	engine.session = base64.RawURLEncoding.EncodeToString([]byte(uuid.String()))

	// make required paths
	os.MkdirAll(conf.Home, os.ModePerm)
	os.MkdirAll(conf.SessionRoot(), os.ModePerm)
	os.MkdirAll(engine.Root(), os.ModePerm)

	// configure resolvers
	engine.resolvers = resolver.NewResolverMap(conf.Resolvers)

	// get lists from the configuration
	lists := conf.Lists

	// load lists (from remote urls)
	for _, list := range lists {
		// get list path
		path := conf.PathToList(list)

		// skip non-remote lists
		if !list.IsRemote() {
			continue
		}

		// skip downloading, don't need to download unless
		// certain conditions are met, which should be triggered
		// from inside the app or similar and not every time
		// an engine is created
		if _, err := os.Stat(path); err == nil {
			continue
		}

		// load/download list if required
		err := downloader.Download(conf, list)
		if err != nil {
			return nil, err
		}
	}

	// empty groups list of size equal to available groups
	workingGroups := append([]*config.GudgeonGroup{}, conf.Groups...)

	// look for default group
	foundDefaultGroup := false
	for _, group := range conf.Groups {
		if "default" == group.Name {
			foundDefaultGroup = true
			break
		}
	}

	// inject default group
	if !foundDefaultGroup {
		defaultGroup := new(config.GudgeonGroup)
		defaultGroup.Name = "default"
		defaultGroup.Tags = []string{"default"}
		workingGroups = append(workingGroups, defaultGroup)
	}

	// use length of working groups to make list of active groups
	groups := make([]*group, len(workingGroups))

	// process groups
	for idx, configGroup := range workingGroups {

		// create active group for gorup name
		engineGroup := new(group)
		engineGroup.engine = engine
		engineGroup.configGroup = configGroup
		// add created engine group to list of groups
		groups[idx] = engineGroup

		// determine which lists belong to this group
		lists := assignedLists(configGroup.Lists, configGroup.Tags, conf.Lists)

		// open the file, read each line, parse to rules
		for _, list := range lists {
			path := conf.PathToList(list)
			array, err := util.GetFileAsArray(path)
			if err != nil {
				continue
			}

			// now parse the array by creating rules and storing them
			parsedType := rule.ParseType(list.Type)
			rules := make([]rule.Rule, len(array))
			for idx, ruleText := range array {
				rules[idx] = rule.CreateRule(ruleText, parsedType)
			}

			// send rule array to engine store
			engine.store.Load(configGroup.Name, rules)
		}

		// clean up after loading all the rules because
		// of all the extra allocation that gets performed
		// during the creation of the arrays and whatnot
		runtime.GC()

		// set default group on engine if found
		if "default" == configGroup.Name {
			engine.defaultGroup = engineGroup
		}
	}

	// attach groups to consumers
	consumers := make([]*consumer, len(conf.Consumers))
	for index, configConsumer := range conf.Consumers {
		// create an active consumer
		consumer := new(consumer)
		consumer.engine = engine
		consumer.groupNames = make([]string, 0)
		consumer.resolverNames = make([]string, 0)
		consumer.configConsumer = configConsumer

		// link consumer to group when the consumer's group elements contains the group name
		for _, group := range groups {
			if util.StringIn(group.configGroup.Name, configConsumer.Groups) {
				consumer.groupNames = append(consumer.groupNames, group.configGroup.Name)

				// add resolvers from group too
				if len(group.configGroup.Resolvers) > 0 {
					consumer.resolverNames = append(consumer.resolverNames, group.configGroup.Resolvers...)
				}
			}
		}

		// add active consumer to list
		consumers[index] = consumer
	}

	// process or clean up consumers

	// set consumers as active on engine
	engine.consumers = consumers

	return engine, nil
}

func (engine *engine) getConsumerForIp(consumerIp net.IP) *consumer {
	var foundConsumer *consumer = nil

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

func (engine *engine) IsDomainBlocked(consumerIp net.IP, domain string) bool {
	// drop ending . if present from domain
	if strings.HasSuffix(domain, ".") {
		domain = domain[:len(domain)-1]
	}

	// get groups applicable to consumer
	groupNames := engine.getConsumerGroups(consumerIp)
	result := engine.store.IsMatchAny(groupNames, domain)
	return !(result == rule.MatchAllow || result == rule.MatchNone)
}

// handles recursive resolution of cnames
func (engine *engine) handleCnameResolution(address net.IP, protocol string, originalRequest *dns.Msg, originalResponse *dns.Msg) *dns.Msg {
	// scope provided finding response
	var response *dns.Msg = nil

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
		cnameResponse := engine.performRequest(address, protocol, cnameRequest)
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

func (engine *engine) performRequest(address net.IP, protocol string, request *dns.Msg) *dns.Msg {
	// scope provided finding response
	var (
		response *dns.Msg                   = nil
		result   *resolver.ResolutionResult = nil
	)
	blocked := false

	// create context
	rCon := resolver.DefaultRequestContext()
	rCon.Protocol = protocol

	// get domain name
	domain := request.Question[0].Name

	// get block status
	if engine.IsDomainBlocked(address, domain) {
		blocked = true

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

	// if no response is found at this point return NXDOMAIN
	if util.IsEmptyResponse(response) {
		response = new(dns.Msg)
		response.SetReply(request)
		response.Rcode = dns.RcodeNameError
	}

	// goroutine log which is async on the other side
	qlog.Log(address, request, response, blocked, rCon, result)

	return response
}

func (engine *engine) Handle(dnsWriter dns.ResponseWriter, request *dns.Msg) {
	// allow us to look up the consumer IP
	var a net.IP = nil

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

	response := engine.performRequest(a, protocol, request)

	// write response to response writer
	dnsWriter.WriteMsg(response)
}
