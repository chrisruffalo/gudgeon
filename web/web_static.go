package web

import (
	"html/template"
	"io/ioutil"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"github.com/chrisruffalo/gudgeon/version"
)

const (
	templateFileExtension = ".tmpl"
	gzipFileExtension = ".gz"
)

// anti-cache from...
// https://stackoverflow.com/a/33881296
var epoch = time.Unix(0, 0).Format(time.RFC1123)

var noCacheHeaders = map[string]string{
	"Expires":         epoch,
	"Cache-Control":   "no-cache, private, max-age=0",
	"Pragma":          "no-cache",
	"X-Accel-Expires": "0",
}

func (web *web) ServeStatic(fs http.FileSystem) gin.HandlerFunc {
	return func(c *gin.Context) {
		url := c.Request.URL

		// dont serve templates
		if strings.HasSuffix(url.Path, templateFileExtension) {
			return
		}

		// for empty path load welcome file (index.html)
		path := url.Path
		if "" == path || "/" == path {
			path = "/index.html"
		}

		// try and open the target file
		file, err := fs.Open(path)

		// if there was an error opening the file or if the file is nil
		// then look for it as a template and if the template is found
		// then process the template
		if err != nil || file == nil {
			// look for .gz (gzipped) file and return that if it exists
			gzipped, err := fs.Open(path + gzipFileExtension)
			if err == nil && gzipped != nil {
				// set gzip output header
				c.Header("Content-Encoding", "gzip")

				// get mime type from extension
				var contentType string
				if strings.Contains(path, ".") {
					ext := path[strings.LastIndex(path, ".") :]
					contentType = mime.TypeByExtension(ext)
				}
				// default to x-gzip if no content type is computed from the extension
				if contentType == "" {
					contentType = "application/x-gzip"
				}

				// get stat for size
				stat, _ := gzipped.Stat()

				// write output
				c.DataFromReader(http.StatusOK, stat.Size(), contentType, gzipped, map[string]string{})

				// close gzipped source
				_ = gzipped.Close()

				// done
				return
			}

			// look for template file and serve it if it exists
			tmpl, err := fs.Open(path + templateFileExtension)
			if err != nil || tmpl == nil {
				// but if it doesn't exist then use the index template
				tmpl, _ = fs.Open("/index.html.tmpl")
			}

			contents, err := ioutil.ReadAll(tmpl)
			if err != nil {
				log.Errorf("Error getting template file contents: %s", err)
			} else {
				parsedTemplate, err := template.New(path).Parse(string(contents))
				if err != nil {
					log.Errorf("Error parsing template file: %s", err)
				}

				// hash
				options := make(map[string]interface{}, 0)
				options["version"] = version.Info()
				options["query_log"] = web.conf.QueryLog.Enabled
				options["query_log_persist"] = web.conf.QueryLog.Persist
				options["metrics"] = web.conf.Metrics.Enabled
				options["metrics_persist"] = web.conf.Metrics.Persist
				options["metrics_detailed"] = web.conf.Metrics.Detailed

				// execute and write template
				c.Status(http.StatusOK)
				err = parsedTemplate.Execute(c.Writer, options)
				if err != nil {
					c.Status(http.StatusInternalServerError)
					log.Errorf("Error executing template: %s", err)
				} else {
					// only return if no error encountered
					return
				}
			}
			return
		}

		// done with file at this point
		stat, _ := file.Stat()

		// get mime type from extension
		var contentType string
		if strings.Contains(path, ".") {
			ext := path[strings.LastIndex(path, ".") :]
			contentType = mime.TypeByExtension(ext)
		}
		// default to application/octet-stream
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		c.DataFromReader(http.StatusOK, stat.Size(), contentType, file, noCacheHeaders)

		// close file
		_ = file.Close()
	}
}
