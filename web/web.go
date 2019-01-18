package web

import (
    "fmt"
    "net/http"

    "github.com/GeertJohan/go.rice"

    "github.com/chrisruffalo/gudgeon/config"
)

type web struct {

}

type Web interface {
    Serve(conf *config.GudgeonConfig) error
}

func New() Web {
    return &web{}
}

func (web *web) Serve(conf *config.GudgeonConfig) error {

    // attach to static assets
    http.Handle("/", http.FileServer(rice.MustFindBox("assets").HTTPBox()))

    // go serve
    webConf := conf.Web
    go http.ListenAndServe(fmt.Sprintf("%s:%d", webConf.Address, webConf.Port), nil)
    fmt.Printf("Started web ui on %s:%d\n", webConf.Address, webConf.Port)

    return nil
}