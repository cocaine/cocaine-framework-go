package cocaine12

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseLocators(t *testing.T) {
	locatorsV1 := "host1:10053,127.0.0.1:10054,ff:fdf::fdfd:10054"
	assert.Equal(t, []string{"host1:10053", "127.0.0.1:10054", "ff:fdf::fdfd:10054"}, parseLocators(locatorsV1))
	locatorsV0 := "localhost:10053"
	assert.Equal(t, []string{"localhost:10053"}, parseLocators(locatorsV0))
}

func TestParseArgs(t *testing.T) {
	args := []string{"--locator", "host1:10053,127.0.0.1:10054",
		"--uuid", "uuid", "--protocol", "1",
		"--endpoint", "/var/run/cocaine/sock"}
	def := newDefaults(args, "test")
	assert.Equal(t, 1, def.Protocol(), "invalid protocol version")
	assert.Equal(t, "uuid", def.UUID(), "invalid uuid")
	assert.Equal(t, "/var/run/cocaine/sock", def.Endpoint(), "invalid endpoint")
	assert.Equal(t, []string{"host1:10053", "127.0.0.1:10054"}, def.Locators(), "invalid locators")
}

func TestParseArgsWithoutLocators(t *testing.T) {
	args := []string{}
	def := newDefaults(args, "test")
	assert.Equal(t, 0, def.Protocol(), "invalid protocol version")
	// assert.Equal(t, "uuid", def.UUID(), "invalid uuid")
	// assert.Equal(t, "/var/run/cocaine/sock", def.Endpoint(), "invalid endpoint")
	assert.Equal(t, []string{"localhost:10053"}, def.Locators(), "invalid locators")
}
