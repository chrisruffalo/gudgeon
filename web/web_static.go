package web

import (
	"html/template"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"github.com/chrisruffalo/gudgeon/version"
)

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

		if file, err := fs.Open(path); err != nil || file == nil {
			// look for template file and serve it if it exists
			if tmpl, err := fs.Open(path + templateFileExtension); err == nil {
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
					options["metrics"] = web.conf.Metrics.Enabled

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
			}
			return
		} else {
			file.Close()
		}

		fileServer.ServeHTTP(c.Writer, c.Request)
	}
}
