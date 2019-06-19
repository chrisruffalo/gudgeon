package resolver

import (
	"github.com/chrisruffalo/gudgeon/util"
	"github.com/miekg/dns"

	log "github.com/sirupsen/logrus"

)

var localServers = []string{
	"127.0.0.1",
	"::1",
	"::",
}

type resolvSource struct {
	// the original file
	filePath      string

	// search domains
	searchDomains []string

	// upstream multisource (or load balanced source)
	upstream      Source
}

func (resolvSource *resolvSource) Load(resolvFilePath string) {
	// save source path
	resolvSource.filePath = resolvFilePath

	// read source file
	config, err := dns.ClientConfigFromFile(resolvFilePath)
	if err != nil {
		log.Errorf("Could not load resolv-style configuration file '%s': %s", resolvFilePath, err)
		return
	}

	// get search domains
	resolvSource.searchDomains = config.Search

	// parse each source
	sources := make([]Source, 0)
	for _, server := range config.Servers {
		// ignore local sources
		if util.StringIn(server, localServers) {
			continue
		}

		// append source
		sources = append(sources, NewSource(server))
	}

	// create source based some multisource for upstream
	if len(sources) > 1 {
		resolvSource.upstream = newMultiSource(resolvFilePath, sources)
	} else if len(sources) == 1 {
		resolvSource.upstream = sources[0]
	}
}


func (resolvSource *resolvSource) Name() string {
	return resolvSource.filePath
}

func (resolvSource *resolvSource) Answer(rCon *RequestContext, context *ResolutionContext, request *dns.Msg) (*dns.Msg, error) {
	// todo: implement search domains
	if resolvSource.upstream != nil {
		resp, err := resolvSource.upstream.Answer(rCon, context, request)
		if !util.IsEmptyResponse(resp) && context != nil {
			context.SourceUsed = resolvSource.Name()
		}
		return resp, err
	}
	return nil, nil
}

func (resolvSource *resolvSource) Close() {

}

