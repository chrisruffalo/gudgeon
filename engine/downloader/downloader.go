package downloader

import (
	"os"
	"net/http"
	"path"
	"bufio"
	"github.com/chrisruffalo/gudgeon/config"
)

func downloadFile(path string, url string) (uint64, error) {
	lines := uint64(0)

	out, err := os.Create(path)
	if err != nil {
        return 0, err
	}
	defer out.Close()

    resp, err := http.Get(url)
    if err != nil {
        return 0, err
    }
    defer resp.Body.Close()

    // Write the body to file line by line so we can count lines
    scanner := bufio.NewScanner(resp.Body)
    for scanner.Scan() {
    	out.Write([]byte(scanner.Text() + "\n"))
    	lines++
    }
    
    return lines, nil
}

func Download(cachePath string, lists []config.GudgeonRemoteList) (uint64, error) {
	// size of bytes copied
	lines := uint64(0)

	// go through lists
	for _, list := range lists {

		// create on-disk name of list
		name := list.Name
		path := path.Join(cachePath, name + ".list")

		// get rul of list
		url := list.URL

		// get written lines
		written, err := downloadFile(path, url)
		if err != nil {
			return lines, err
		}
		lines += written 
	}


	return lines, nil
}