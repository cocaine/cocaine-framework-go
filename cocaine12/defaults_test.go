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
