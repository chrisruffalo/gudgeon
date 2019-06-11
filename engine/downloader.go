package engine

import (
	"context"
	"net"
	"net/http"
	"os"
	paths "path"
	"strings"
	"time"

	"github.com/cavaliercoder/grab"
	log "github.com/sirupsen/logrus"

	"github.com/chrisruffalo/gudgeon/config"
)

func downloadFile(engine Engine, path string, url string) error {
	// don't do anything with empty url
	if url == "" {
		return nil
	}

	dirpart := paths.Dir(path)
	if _, err := os.Stat(dirpart); os.IsNotExist(err) {
		err := os.MkdirAll(dirpart, os.ModePerm)
		if err != nil {
			log.Errorf("Could not create path to download file: %s", err)
		}
	}

	// set up (default) http client
	client := &http.Client{}

	// if the engine is available then use it to resolve hostnames
	if engine != nil {
		// create dialer
		dialer := &net.Dialer{
			KeepAlive: 5 * time.Second,
		}

		// create transport
		tr := &http.Transport{}

		// update the dial context in the client which allows us to replace the dialed address
		// with the one that we create here
		tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			// manually resolve address if it is not an ip
			split := strings.Split(addr, ":")
			if len(split) > 1 && net.ParseIP(split[0]) == nil {
				resolvedIP, err := engine.Resolve(split[0])
				if err == nil && "" != resolvedIP {
					addr = resolvedIP + ":" + split[1]
				}
			}
			// chain the dialer into the default context either using the new address or the original address if no resolution happened
			return dialer.DialContext(ctx, network, addr)
		}

		// set transport on client
		client.Transport = tr
	}

	// use the http client to make a grabber client
	grabber := grab.Client{
		HTTPClient: client,
	}
	req, err := grab.NewRequest(path, url)
	if err != nil {
		return err
	}
	resp := grabber.Do(req)
	err = resp.Err()
	if err != nil {
		return err
	}

	return nil
}

func Download(engine Engine, config *config.GudgeonConfig, list *config.GudgeonList) error {
	// create on-disk name of list
	path := config.PathToList(list)

	// get rul of list
	url := list.Source

	// notify of download and save to path
	log.Infof("Downloading '%s'...", url)
	err := downloadFile(engine, path, url)
	if err != nil {
		return err
	}

	return nil
}
