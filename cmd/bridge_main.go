package main

import (
	"log"
	"os"

	cocaine "github.com/cocaine/cocaine-framework-go/cocaine12"
	"github.com/cocaine/cocaine-framework-go/cocaine12/bridge"
)

func main() {
	logger, err := cocaine.NewLogger()
	if err != nil {
		log.Fatalf("unable to create logger %v", err)
	}
	defer logger.Close()

	cfg := bridge.DefaultBridgeConfig()

	name := os.Getenv("slave")
	cfg.Name = name

	b, err := bridge.NewBridge(cfg, logger)
	if err != nil {
		log.Fatalf("unable to create a bridge %v", err)
	}
	defer b.Close()

	if err := b.Start(); err != nil {
		log.Fatalf("bridge stopped with an error: %v", err)
	}
}
