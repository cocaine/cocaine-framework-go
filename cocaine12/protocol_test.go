package cocaine12

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEndpoint(t *testing.T) {
	e := EndpointItem{
		IP:   "127.0.0.1",
		Port: 10053,
	}

	assert.Equal(t, "127.0.0.1:10053", e.String())
}
