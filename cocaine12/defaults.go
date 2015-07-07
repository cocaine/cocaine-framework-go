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
	Debug    bool
}

var defaults = newDeafults(os.Args[1:], "cocaine")

func parseLocators(arg string) []string {
	if strings.IndexRune(arg, ',') == -1 {
		return []string{arg}
	}

	return strings.Split(arg, ",")
}

func newDeafults(args []string, setname string) *defaultsValues {
	values := new(defaultsValues)
	var locators string

	flagSet := flag.NewFlagSet(setname, flag.ContinueOnError)
	flagSet.StringVar(&values.AppName, "app", "gostandalone", "application name")
	flagSet.StringVar(&values.Endpoint, "endpoint", "", "unix socket path to connect to the Cocaine")
	flagSet.StringVar(&locators, "locator", "localhost:10053", "default endpoints of locators")
	flagSet.IntVar(&values.Protocol, "protocol", 0, "protocol version")
	flagSet.StringVar(&values.UUID, "uuid", "", "UUID")

	flagSet.Parse(args)
	values.Locators = parseLocators(locators)

	values.Debug = strings.ToUpper(os.Getenv("DEBUG")) == "DEBUG"

	return values
}
