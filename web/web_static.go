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

func Serve(fs http.FileSystem) gin.HandlerFunc {
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

                    // execute and write template
                    c.Status(http.StatusOK)
                    err = parsedTemplate.Execute(c.Writer, version.Info())
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