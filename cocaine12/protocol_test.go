package cocaine12

import (
	"bytes"
	"sort"
	"testing"

	"github.com/tinylib/msgp/msgp"
	"github.com/ugorji/go/codec"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEndpoint(t *testing.T) {
	e := EndpointItem{
		IP:   "127.0.0.1",
		Port: 10053,
	}

	assert.Equal(t, "127.0.0.1:10053", e.String())
}

func TestAPIUnpack(t *testing.T) {
	// {0: ['resolve', {}, {0: ['value', {}], 1: ['error', {}]}],
	//  1: ['connect', {}, {0: ['write', None], 1: ['error', {}], 2: ['close', {}]}],
	//  2: ['refresh', {}, {0: ['value', {}], 1: ['error', {}]}],
	//  3: ['cluster', {}, {0: ['value', {}], 1: ['error', {}]}],
	//  4: ['publish', {0: ['discard', {}]}, {0: ['value', {}], 1: ['error', {}]}],
	//  5: ['routing', {}, {0: ['write', None], 1: ['error', {}], 2: ['close', {}]}]}
	payload := []byte{134, 0, 147, 167, 114, 101, 115, 111, 108, 118, 101, 128, 130, 0, 146, 165, 118, 97, 108, 117, 101, 128,
		1, 146, 165, 101, 114, 114, 111, 114, 128, 1, 147, 167, 99, 111, 110, 110, 101, 99, 116, 128, 131, 0, 146,
		165, 119, 114, 105, 116, 101, 192, 1, 146, 165, 101, 114, 114, 111, 114, 128, 2, 146, 165, 99, 108, 111, 115,
		101, 128, 2, 147, 167, 114, 101, 102, 114, 101, 115, 104, 128, 130, 0, 146, 165, 118, 97, 108, 117, 101, 128, 1,
		146, 165, 101, 114, 114, 111, 114, 128, 3, 147, 167, 99, 108, 117, 115, 116, 101, 114, 128, 130, 0, 146, 165, 118,
		97, 108, 117, 101, 128, 1, 146, 165, 101, 114, 114, 111, 114, 128, 4, 147, 167, 112, 117, 98, 108, 105, 115, 104,
		129, 0, 146, 167, 100, 105, 115, 99, 97, 114, 100, 128, 130, 0, 146, 165, 118, 97, 108, 117, 101, 128, 1, 146, 165,
		101, 114, 114, 111, 114, 128, 5, 147, 167, 114, 111, 117, 116, 105, 110, 103, 128, 131, 0, 146, 165, 119, 114, 105,
		116, 101, 192, 1, 146, 165, 101, 114, 114, 111, 114, 128, 2, 146, 165, 99, 108, 111, 115, 101, 128}

	var dm dispatchMap
	decoder := codec.NewDecoder(bytes.NewReader(payload), hAsocket)
	err := decoder.Decode(&dm)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	expected := []string{"resolve", "connect", "refresh", "cluster", "publish", "routing"}
	sort.Strings(expected)
	actual := dm.Methods()
	sort.Strings(actual)
	assert.Equal(t, expected, actual)

	for _, v := range dm {
		if v.Name == "connect" {
			assert.True(t, v.Downstream.Type() == emptyDispatch)
			assert.True(t, v.Upstream.Type() == otherDispatch)

			for _, descrItem := range *v.Upstream {
				switch descrItem.Name {
				case "write":
					assert.True(t, descrItem.Description.Type() == recursiveDispatch)
				case "close":
					assert.True(t, descrItem.Description.Type() == emptyDispatch)
				case "error":
					assert.True(t, descrItem.Description.Type() == emptyDispatch)
				}
			}
		}
	}

}

func TestMessageEncodeDecode(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	// Payload is packed by Python:
	// '[]byte{'+ ', '.join([str(ord(i)) for i in msgpack.packb([1, 0, ["A", "B", "C", 100, 10.],
	// [100, [False, "trace_id", '|\x00\x00\x00\x00\x00\x00\x00'], [True, 101, ""],
	// [False, "span_id", '\x9f\x86\x01\x00\x00\x00\x00\x00']]])]) + '}'
	payload := []byte{148, 1, 0, 149, 161, 65, 161, 66, 161, 67, 100, 203, 64, 36, 0, 0, 0, 0, 0, 0,
		148, 100, 147, 194, 168, 116, 114, 97, 99, 101, 95, 105, 100, 168, 124,
		0, 0, 0, 0, 0, 0, 0, 147, 195, 101, 160, 147, 194, 167, 115, 112, 97,
		110, 95, 105, 100, 168, 159, 134, 1, 0, 0, 0, 0, 0}
	var msg message
	require.NoError(msg.DecodeMsg(msgp.NewReader(bytes.NewReader(payload))), "DecodeMsg returns non-nil error")
	assert.Equal(uint64(1), msg.session, "invalid message session")
	assert.Equal(uint64(0), msg.msgType, "invalid message type")
	headers := msg.headers
	// 2 malformed headers are skipped
	assert.Equal(2, len(headers), "invalid headers length")

	var buff = new(bytes.Buffer)
	assert.NoError(msgp.Encode(buff, &msg), "EncodeMsg returns non-nil error")
	// 20 - length w/o headers
	assert.Equal(payload[:20], buff.Bytes()[:20], "EncodeMsg diffs from etalon data")
}

func TestHeadersTraceData(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	trace := TraceInfo{
		trace:  uint64(100),
		span:   uint64(200),
		parent: uint64(300),
	}

	headers := make(CocaineHeaders)
	traceInfoToHeaders(headers, &trace)
	assert.Equal(3, len(headers))

	traceInfo, err := headers.getTraceData()
	require.NoError(err)

	assert.Equal(trace.trace, traceInfo.trace)
	assert.Equal(trace.span, traceInfo.span)
	assert.Equal(trace.parent, traceInfo.parent)
}
