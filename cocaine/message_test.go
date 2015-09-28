package cocaine

import (
	"testing"

	"github.com/cocaine/cocaine-framework-go/vendor/src/github.com/ugorji/go/codec"
	"github.com/stretchr/testify/assert"
)

func TestMessageUnpackErrorMsg(t *testing.T) {
	const (
		session      = 1
		_type        = 5
		errorCode    = 1
		errorMessage = "someerror"
	)
	// msgpack.packb([5, 1, [1, "someerror"]])
	payload := []byte{147, 5, 1, 146, 1, 169, 115, 111, 109, 101, 101, 114, 114, 111, 114}
	var result []interface{}

	assert.NoError(t, codec.NewDecoderBytes(payload, h).Decode(&result))

	msg, err := unpackMessage(result)
	assert.NoError(t, err)

	assert.Equal(t, session, msg.getSessionID(), "bad session")
	if !assert.Equal(t, _type, msg.getTypeID(), "bad message type") {
		t.FailNow()
	}

	e := msg.(*errorMsg)
	assert.Equal(t, errorCode, e.Code, "bad error code")
	assert.Equal(t, errorMessage, e.Message, "bad error message")
}
