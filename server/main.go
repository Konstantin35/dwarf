package main

import (
	"encoding/json"

	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/yvasiyarov/gorelic"

    log "github.com/dmuth/google-go-log4go"

	"bitbucket.org/vdidenko/dwarf/server/api"
	"bitbucket.org/vdidenko/dwarf/server/payouts"
	"bitbucket.org/vdidenko/dwarf/server/proxy"
	"bitbucket.org/vdidenko/dwarf/server/storage"
)

var cfg proxy.Config
var backend *storage.RedisClient

func startProxy() {
	s := proxy.NewProxy(&cfg, backend)
	s.Start()
}

func startApi() {
	s := api.NewApiServer(&cfg.Api, backend)
	s.Start()
}

func startBlockUnlocker() {
	u := payouts.NewBlockUnlocker(&cfg.BlockUnlocker, backend)
	u.Start()
}

func startPayoutsProcessor() {
	u := payouts.NewPayoutsProcessor(&cfg.Payouts, backend)
	u.Start()
}

func startNewrelic() {
	if cfg.NewrelicEnabled {
		nr := gorelic.NewAgent()
		nr.Verbose = cfg.NewrelicVerbose
		nr.NewrelicLicense = cfg.NewrelicKey
		nr.NewrelicName = cfg.NewrelicName
		nr.Run()
	}
}

func readConfig(cfg *proxy.Config) {
	configFileName := "config.json"
	if len(os.Args) > 1 {
		configFileName = os.Args[1]
	}
	configFileName, _ = filepath.Abs(configFileName)

	log.Infof("Loading config: %v", configFileName)

	configFile, err := os.Open(configFileName)
	if err != nil {
		log.Errorf("File error: ", err.Error())
	}
	defer configFile.Close()
	jsonParser := json.NewDecoder(configFile)
	if err := jsonParser.Decode(&cfg); err != nil {
		log.Errorf("Config error: ", err.Error())
	}
}

func main() {
	log.SetLevel(log.InfoLevel)
	log.SetDisplayTime(true)
	readConfig(&cfg)
	rand.Seed(time.Now().UnixNano())

	if cfg.Threads > 0 {
		runtime.GOMAXPROCS(cfg.Threads)
		log.Infof("Running with %v threads", cfg.Threads)
	}

	startNewrelic()

	backend = storage.NewRedisClient(&cfg.Redis, cfg.Coin)
	pong, err := backend.Check()
	if err != nil {
		log.Infof("Can't establish connection to backend: %v", err)
	} else {
		log.Infof("Backend check reply: %v", pong)
	}

	if cfg.Proxy.Enabled {
		go startProxy()
	}
	if cfg.Api.Enabled {
		go startApi()
	}
	if cfg.BlockUnlocker.Enabled {
		go startBlockUnlocker()
	}
	if cfg.Payouts.Enabled {
		go startPayoutsProcessor()
	}
	quit := make(chan bool)
	<-quit
}
