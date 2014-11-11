package cocaine

import (
	"github.com/ugorji/go/codec"
)

// It must be done other way
func convertPayload(in interface{}, out interface{}) error {
	var buf []byte
	if err := codec.NewEncoderBytes(&buf, h).Encode(in); err != nil {
		return err
	}
	if err := codec.NewDecoderBytes(buf, h).Decode(out); err != nil {
		return err
	}

	return nil
}
