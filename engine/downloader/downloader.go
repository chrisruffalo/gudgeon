package downloader

import (
	"os"
	"net/http"
	"bufio"
	"github.com/chrisruffalo/gudgeon/config"
)

func downloadFile(path string, url string) (uint, error) {
	lines := uint(0)

	// if the file exists already then it might be used during the
	// download process so we need to move it
	if _, err := os.Stat(path); err == nil {
		// download file to temp location
		inactivePath := path + "_inactive"
		lines, err := downloadFile(inactivePath, url)
		if err != nil {
			return 0, err
		}

		// move file over existing file when done
		os.Rename(inactivePath, path)	

		// complete action
		return lines, nil
	}

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

func Download(config *config.GudgeonConfig, list config.GudgeonRemoteList) (uint, error) {
	// create on-disk name of list
	path := config.PathToList(list)

	// get rul of list
	url := list.URL

	// get written lines
	written, err := downloadFile(path, url)
	if err != nil {
		return 0, err
	}

	return written, nil
}

func DownloadAll(config *config.GudgeonConfig) (uint, error) {
	// size of bytes copied
	lines := uint(0)

	// go through lists
	for _, list := range config.Blocklists {
		written, err := Download(config, list)
		if err != nil {
			return lines, err
		}
		lines += written
	}

	return lines, nil
}