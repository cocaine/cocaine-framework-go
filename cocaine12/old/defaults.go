package cocaine

import (
	"flag"
)

var (
	DefaultLocator string = ":10053"
	DefaultAppName string = "standalone"
)

var (
	flagUUID     string
	flagEndpoint string
)

func init() {
	flag.StringVar(&flagUUID, "uuid", "", "UUID")
	flag.StringVar(&flagEndpoint, "endpoint", "", "Connection path")
	flag.StringVar(&DefaultAppName, "app", "standalone", "Connection path")
	flag.StringVar(&DefaultLocator, "locator", "localhost:10053", "Connection path")
	flag.Parse()
}
