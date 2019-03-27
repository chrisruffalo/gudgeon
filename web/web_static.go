package web

import (
	"html/template"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"github.com/chrisruffalo/gudgeon/version"
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

var etagHeaders = []string{
	"ETag",
	"If-Modified-Since",
	"If-Match",
	"If-None-Match",
	"If-Range",
	"If-Unmodified-Since",
}

func (web *web) ServeStatic(fs http.FileSystem) gin.HandlerFunc {
	fileServer := http.StripPrefix("/", http.FileServer(fs))
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
				defer tmpl.Close()
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
		file.Close()

		// remove etag headers (don't request caching)
		for _, v := range etagHeaders {
			if c.Request.Header.Get(v) != "" {
				c.Request.Header.Del(v)
			}
		}

		// serve
		fileServer.ServeHTTP(c.Writer, c.Request)

		// strip etags (don't cache this stuff)
		for k, v := range noCacheHeaders {
			c.Writer.Header().Set(k, v)
		}
	}
}
