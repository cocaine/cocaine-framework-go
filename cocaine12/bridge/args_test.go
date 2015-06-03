package bridge

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterArgs(t *testing.T) {
	var input = []string{
		"--locator", "host1:10053,127.0.0.1:10054",
		"--uuid", "uuid",
		"--endpoint", "/var/run/cocaine/sock",
		"--protocol", "1"}

	result := filterEndpointArg(input)
	assert.Equal(t, []string{"--locator", "host1:10053,127.0.0.1:10054", "--uuid", "uuid", "--protocol", "1"}, result)

	var input2 = []string{
		"--locator", "host1:10053,127.0.0.1:10054",
		"--uuid", "uuid",
		"--protocol", "1",
		"--endpoint", "/var/run/cocaine/sock"}
	result = filterEndpointArg(input2)
	assert.Equal(t, []string{"--locator", "host1:10053,127.0.0.1:10054", "--uuid", "uuid", "--protocol", "1"}, result)
}
