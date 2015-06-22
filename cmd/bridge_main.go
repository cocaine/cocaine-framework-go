package main

import (
	"log"
	"os"
	"strconv"

	cocaine "github.com/cocaine/cocaine-framework-go/cocaine12"
	"github.com/cocaine/cocaine-framework-go/cocaine12/bridge"
	"github.com/cocaine/cocaine-framework-go/version"
)

func main() {
	logger, err := cocaine.NewLogger()
	if err != nil {
		log.Fatalf("unable to create logger %v", err)
	}
	defer logger.Close()

	cfg := bridge.NewBridgeConfig()
	if cfg.Name = os.Getenv("slave"); cfg.Name == "" {
		logger.WithFields(
			cocaine.Fields{
				"appsource":     "bridge",
				"bridgeversion": version.Version,
			}).Err("unable to determine a path to the slave")
		// to have it in crashlogs
		log.Fatal("unable to determine a path to the slave")
	}

	if strport := os.Getenv("port"); strport != "" {
		// the port is specified
		cfg.Port, err = strconv.Atoi(strport)
		if err != nil {
			logger.WithFields(
				cocaine.Fields{
					"appsource":     "bridge",
					"bridgeversion": version.Version,
				}).Errf("unable to determine the port %v", err)
			// to have it in crashlogs
			log.Fatalf("unable to determine the port %v", err)
		}
	}

	if strStartupTimeout := os.Getenv("startup-timeout"); strStartupTimeout != "" {
		// the port is specified
		cfg.StartupTimeout, err = strconv.Atoi(strStartupTimeout)
		if err != nil {
			logger.WithFields(
				cocaine.Fields{
					"appsource":     "bridge",
					"bridgeversion": version.Version,
				}).Errf("unable to determine the startup timeout %v", err)
			// to have it in crashlogs
			log.Fatalf("unable to determine the startup timeout %v", err)
		}
	}

	logger.WithFields(cocaine.Fields{
		"appsource":      "bridge",
		"bridgeversion":  version.Version,
		"slave":          cfg.Name,
		"port":           cfg.Port,
		"startuptimeout": cfg.StartupTimeout,
	}).Infof("starting new bridge")

	b, err := bridge.NewBridge(cfg, logger)
	if err != nil {
		logger.WithFields(
			cocaine.Fields{
				"appsource":     "bridge",
				"bridgeversion": version.Version,
			}).Errf("unable to create a bridge %v", err)
		log.Fatalf("unable to create bridge %v", err)
	}
	defer b.Close()

	if err := b.Start(); err != nil {
		logger.WithFields(
			cocaine.Fields{
				"appsource":    "bridge",
				"bridgeverson": version.Version,
			}).Errf("bridge returned with error: %v", err)
	}
}
