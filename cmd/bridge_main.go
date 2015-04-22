package main

import (
	"github.com/cocaine/cocaine-framework-go/cocaine12/bridge"
)

func main() {
	cfg := bridge.DefaultBridgeConfig()
	b, err := bridge.NewBridge(cfg)
	if err != nil {
		panic(err)
	}
	defer b.Close()
	b.Start()
}
