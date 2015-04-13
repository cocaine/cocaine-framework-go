package cocaine12

import (
	"flag"
)

var (
	// DefaultLocator is used by Locator to resolve services by default
	DefaultLocator = ":10053"

	defaultAppName = "gostandalone"

	flagUUID     string
	flagEndpoint string
)

func setupFlags() {
	flag.StringVar(&flagUUID, "uuid", "", "UUID")
	flag.StringVar(&flagEndpoint, "endpoint", "", "Connection path")
	flag.StringVar(&defaultAppName, "app", "standalone", "Connection path")
	flag.StringVar(&DefaultLocator, "locator", "localhost:10053", "Connection path")
}
