package engine

import (
	"context"
	"io"
	"net"
	"net/http"
	urls "net/url"
	"os"
	paths "path"
	"strings"
	"time"

	"github.com/chrisruffalo/gudgeon/config"
)

func downloadFile(engine Engine, path string, url string) error {
	// don't do anything with empty url
	if url == "" {
		return nil
	}

	dirpart := paths.Dir(path)
	if _, err := os.Stat(dirpart); os.IsNotExist(err) {
		os.MkdirAll(dirpart, os.ModePerm)
	}

	// if the file exists already then it might be used during the
	// download process so we need to move it
	if _, err := os.Stat(path); err == nil {
		// download file to temp location
		inactivePath := path + "_inactive"
		err = downloadFile(engine, inactivePath, url)
		if err != nil {
			return err
		}

		// move file over existing file when done
		os.Rename(inactivePath, path)

		// complete action
		return nil
	}

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	// set up http client
	client := &http.Client{}

	// if we can't resolve the url normally we might need to use the configured resolvers to find it... eek
	if engine != nil {
		// parse out url
		parsedUrl, urlErr := urls.Parse(url)
		if urlErr == nil {
			hostname := parsedUrl.Hostname()

			// if we can resolve the IP we continue with the resolution substitution
			resolvedIP, err := engine.Resolve(hostname)
			if err == nil && "" != resolvedIP {
				// create dialer
			    dialer := &net.Dialer{
			        Timeout:   5 * time.Second,
			        KeepAlive: 5 * time.Second,
			        DualStack: true,
			    }

				// create transport
				tr := &http.Transport {}

				// update the dial context in the client which allows us to replace the dialed address
				// with the one that we create here
				tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
					// replace the dial with the resolved dial
    				if strings.HasPrefix(addr, hostname + ":")  {
        				addr = strings.Replace(addr, hostname, resolvedIP, 1)
    				}
    				return dialer.DialContext(ctx, network, addr)					
    			}

				// set transport on client
				client.Transport = tr
			}
		}
	}

	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
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

	// get written lines
	err := downloadFile(engine, path, url)
	if err != nil {
		return err
	}

	return nil
}
