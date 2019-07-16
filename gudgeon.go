package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/engine"
	"github.com/chrisruffalo/gudgeon/events"
	"github.com/chrisruffalo/gudgeon/provider"
	"github.com/chrisruffalo/gudgeon/util"
	"github.com/chrisruffalo/gudgeon/version"
	"github.com/chrisruffalo/gudgeon/web"
)

// default divider
const divider = "==============================="

// Gudgeon Core Gudgeon object for executing a Gudgeon process
type Gudgeon struct {
	confPath string
	config   *config.GudgeonConfig
	engine   engine.Engine
	provider provider.Provider
	web      web.Web
}

// NewGudgeon Create a new Gudgeon instance from a given Gudgeon Config
func NewGudgeon(confPath *string, config *config.GudgeonConfig) *Gudgeon {
	return &Gudgeon{
		confPath: *confPath,
		config:   config,
	}
}

func (gudgeon *Gudgeon) Start() error {
	// error
	var err error

	// start the event bus
	events.Start()

	// start the file watcher
	events.StartFileWatch()

	// create engine which handles resolution, logging, etc
	gudgeon.engine, err = engine.NewReloadingEngine(gudgeon.confPath, gudgeon.config)
	if err != nil {
		return err
	}
	if gudgeon.engine == nil {
		return fmt.Errorf("Could not create required engine component")
	}

	// create a new provider and start hosting
	gudgeon.provider = provider.NewProvider(gudgeon.engine)
	err = gudgeon.provider.Host(gudgeon.config, gudgeon.engine)
	if err != nil {
		return fmt.Errorf("Could not create provider: %s", err)
	}

	// open web ui if web enabled
	if gudgeon.config.Web.Enabled {
		gudgeon.web = web.New()
		err = gudgeon.web.Serve(gudgeon.config, gudgeon.engine)
		if err != nil {
			return fmt.Errorf("Could not host web: %s", err)
		}
	}

	// try and print out error if we caught one during startup
	if recovery := recover(); recovery != nil {
		return fmt.Errorf("unrecoverable error: %s", recovery)
	}

	return nil
}

func (gudgeon *Gudgeon) Shutdown() {
	// stop the file watcher
	events.StopFileWatch()

	// stop the event bus
	events.Stop()

	// stop providers
	if gudgeon.provider != nil {
		log.Infof("Shutting down DNS endpoints...")
		err := gudgeon.provider.Shutdown()
		if err != nil {
			log.Errorf("Could not shutdown endpoints: %s", err)
		}
	}

	// stop web
	if gudgeon.web != nil {
		log.Infof("Shutting down Web service...")
		gudgeon.web.Stop()
	}

	// stop/shutdown engine
	log.Infof("Shutting down Engine...")
	gudgeon.engine.Shutdown()
}

func main() {
	// set initial log instance configuration
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// load command options
	opts, err := config.Options(version.GetLongVersion())
	if err != nil {
		log.Errorf("%s", err)
		os.Exit(1)
	}

	// debug print config
	log.Info(divider)
	log.Infof("Gudgeon %s", version.GetLongVersion())
	log.Info(divider)

	// start profiling if enabled
	if opts.DebugOptions.Profile {
		// start profile http endpoint on given port
		profPort := "9900"
		go func() {
			err := http.ListenAndServe("127.0.0.1:"+profPort, nil)
			if err != nil {
				log.Errorf("Could not expose HTTP profile endpoint on %s: %s", profPort, err)
			}
		}()
	}

	// load config
	filename := string(opts.AppOptions.ConfigPath)
	conf, warnings, err := config.Load(filename)
	if err != nil {
		log.Errorf("%s", err)
		os.Exit(1)
	}

	// configure log file from configuration if additional configuration is available

	// print log warnings and continue
	if len(warnings) > 0 {
		for _, warn := range warnings {
			log.Warn(warn)
		}
	}

	// print log file information
	if "" != filename {
		log.Infof("Configuration file: %s", filename)
	}

	// create new Gudgeon instance
	instance := NewGudgeon(&filename, conf)

	// start new instance
	err = instance.Start()
	if err != nil {
		log.Errorf("Error starting Gudgeon: %s", err)
		os.Exit(1)
	}

	// wait for signal
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig

	// clean out session directory
	if "" != conf.SessionRoot() {
		util.ClearDirectory(conf.SessionRoot())
	}

	log.Infof("Signal (%s) received, stopping", s)
	// stop gudgeon, hopefully gracefully
	instance.Shutdown()

	// debugging: print any still-running goroutines
	//pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
}
