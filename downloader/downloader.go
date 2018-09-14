package downloader

import (
	"io"
	"os"
	"net/http"

	"github.com/chrisruffalo/gudgeon/config"
)

func downloadFile(path string, url string) error {
	// if the file exists already then it might be used during the
	// download process so we need to move it
	if _, err := os.Stat(path); err == nil {
		// download file to temp location
		inactivePath := path + "_inactive"
		err = downloadFile(inactivePath, url)
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

    resp, err := http.Get(url)
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

func Download(config *config.GudgeonConfig, list *config.GudgeonList) error {
	// create on-disk name of list
	path := config.PathToList(list)

	// get rul of list
	url := list.Source

	// get written lines
	err := downloadFile(path, url)
	if err != nil {
		return err
	}

	return nil
}