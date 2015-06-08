package cocaine12

import (
	"flag"
	"os"
	"strings"
)

type defaultsValues struct {
	AppName  string
	Endpoint string
	Locators []string
	Protocol int
	UUID     string
}

var defaults = newDeafults(os.Args[1:])

func parseLocators(arg string) []string {
	if strings.IndexRune(arg, ',') == -1 {
		return []string{arg}
	}

	return strings.Split(arg, ",")
}

func newDeafults(args []string) *defaultsValues {
	values := new(defaultsValues)
	var locators string

	flagSet := flag.NewFlagSet("cocaine", flag.ContinueOnError)
	flag.StringVar(&values.AppName, "app", "gostandalone", "application name")
	flag.StringVar(&values.Endpoint, "endpoint", "", "unix socket path to connect to the Cocaine")
	flag.StringVar(&locators, "locator", "localhost:10053", "default endpoints of locators")
	flag.IntVar(&values.Protocol, "protocol", 0, "protocol version")
	flag.StringVar(&values.UUID, "uuid", "", "UUID")

	flagSet.Parse(args)
	values.Locators = parseLocators(locators)

	return values
}
