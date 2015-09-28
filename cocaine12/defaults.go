package cocaine12

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
)

const (
	defaultProtocolVersion = 0
	defaultLocatorEndpoint = "localhost:10053"
)

type defaultValues struct {
	appName  string
	endpoint string
	locators locatorsType
	protocol int
	uuid     string
	debug    bool
}

func (d *defaultValues) ApplicationName() string {
	return d.appName
}

func (d *defaultValues) Endpoint() string {
	return d.endpoint
}

func (d *defaultValues) Debug() bool {
	return d.debug
}

func (d *defaultValues) Locators() []string {
	return d.locators
}

func (d *defaultValues) Protocol() int {
	return d.protocol
}

func (d *defaultValues) UUID() string {
	return d.uuid
}

// DefaultValues provides an interface to read
//various information provided by Cocaine-Runtime to the worker
type DefaultValues interface {
	ApplicationName() string
	Debug() bool
	Endpoint() string
	Locators() []string
	Protocol() int
	UUID() string
}

var (
	initDefaultValues sync.Once
	storedDefaults    DefaultValues

	parseDefaultValues = func() {
		storedDefaults = newDeafults(os.Args[1:], "cocaine")
	}
)

// GetDefaults returns DefaultValues
func GetDefaults() DefaultValues {
	// lazy init
	initDefaultValues.Do(parseDefaultValues)

	return storedDefaults
}

type locatorsType []string

func (l *locatorsType) Set(value string) error {
	(*l) = parseLocators(value)
	return nil
}

func (l *locatorsType) String() string {
	return strings.Join((*l), ",")
}

func parseLocators(arg string) []string {
	if strings.IndexRune(arg, ',') == -1 {
		return []string{arg}
	}

	return strings.Split(arg, ",")
}

func newDeafults(args []string, setname string) *defaultValues {
	var (
		values = new(defaultValues)

		showVersion bool
	)

	values.locators = []string{defaultLocatorEndpoint}
	values.debug = strings.ToUpper(os.Getenv("DEBUG")) == "DEBUG"

	flagSet := flag.NewFlagSet(setname, flag.ContinueOnError)
	flagSet.SetOutput(ioutil.Discard)
	flagSet.StringVar(&values.appName, "app", "gostandalone", "application name")
	flagSet.StringVar(&values.endpoint, "endpoint", "", "unix socket path to connect to the Cocaine")
	flagSet.Var(&values.locators, "locator", "default endpoints of locators")
	flagSet.IntVar(&values.protocol, "protocol", defaultProtocolVersion, "protocol version")
	flagSet.StringVar(&values.uuid, "uuid", "", "UUID")
	flagSet.BoolVar(&showVersion, "showcocaineversion", false, "print framework version")
	flagSet.Parse(args)

	if showVersion {
		fmt.Fprintf(os.Stderr, "Built with Cocaine framework %s\n", frameworkVersion)
		os.Exit(0)
	}

	return values
}
