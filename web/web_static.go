package web

import (
	rice "github.com/GeertJohan/go.rice"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"html/template"
	"io/ioutil"
	"mime"
	"net/http"
	"strings"

	"github.com/chrisruffalo/gudgeon/version"
)

const (
	templateFileExtension = ".tmpl"
	gzipFileExtension     = ".gz"
)

func (web *web) ServeStatic(fs *rice.Box) gin.HandlerFunc {
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
					ext := path[strings.LastIndex(path, "."):]
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

				// todo: this might not work if the template has already started writing we can't change the status code
				if err != nil {
					c.Status(http.StatusInternalServerError)
					log.Errorf("Error executing template: %s", err)
				}
			}
			return
		}

		// done with file at this point
		stat, _ := file.Stat()

		// get mime type from extension
		var contentType string
		if strings.Contains(path, ".") {
			ext := path[strings.LastIndex(path, "."):]
			contentType = mime.TypeByExtension(ext)
		}
		// default to application/octet-stream
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		// write file
		c.DataFromReader(http.StatusOK, stat.Size(), contentType, file, nil)

		// close file
		_ = file.Close()
	}
}
