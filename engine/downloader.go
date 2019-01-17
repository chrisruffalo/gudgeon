package engine

import (
	"context"
	"io"
	"net"
	"net/http"
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
		// create dialer
		dialer := &net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 5 * time.Second,
			DualStack: true,
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
