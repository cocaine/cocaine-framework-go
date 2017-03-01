package cocaine12

import (
	"bytes"
	"sort"
	"testing"

	"github.com/tinylib/msgp/msgp"
	"github.com/ugorji/go/codec"

	"github.com/stretchr/testify/assert"
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
				t.Log(descrItem)
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

//'[]byte{148,1,0,147,161,65,161,66,161,67,147,147,194,80,168,124,0,0,0,0,0,0,0,147,194,81,168,159,134,1,0,0,0,0,0,82}'
func TestMessageEncodeDecode(t *testing.T) {
	assrt := assert.New(t)
	// Payload is packed by Python:
	// traceid = 124
	// parentid = 0
	// spanid=999999
	// [(False, 80, '|\x00\x00\x00\x00\x00\x00\x00'),
	// (False, 81, '\x9f\x86\x01\x00\x00\x00\x00\x00'),
	// 82]
	// [1, 0, ["A", "B", "C"], z]
	payload := []byte{148, 100, 101, 147, 161, 65, 161, 66, 161, 67, 147, 147, 194, 80,
		168, 124, 0, 0, 0, 0, 0, 0, 0, 147, 194, 81, 168, 159, 134, 1, 0, 0, 0, 0, 0, 82}
	var msg message
	assrt.NoError(msg.DecodeMsg(msgp.NewReader(bytes.NewReader(payload))), "DecodeMsg returns non-nil error")
	assrt.Equal(uint64(100), msg.session, "invalid message session")
	assrt.Equal(uint64(101), msg.msgType, "invalid message type")
	headers := msg.headers
	assrt.Equal(3, len(headers), "invalid headers length")

	var buff = new(bytes.Buffer)
	w := msgp.NewWriter(buff)
	assrt.NoError(msg.EncodeMsg(w), "EncodeMsg returns non-nil error")
	w.Flush()
	assrt.Equal(payload, buff.Bytes(), "EncodeMsg diffs from etalon data")
}

// func TestHeaders(t *testing.T) {
// 	var (
// 		//trace.pack_trace(trace.Trace(traceid=9000, spanid=11000, parentid=8000))
// 		buff = []byte{
// 			147, 147, 194, 80, 168, 40, 35, 0, 0, 0, 0, 0, 0, 147, 194, 81, 168,
// 			248, 42, 0, 0, 0, 0, 0, 0, 147, 194, 82, 168, 64, 31, 0, 0, 0, 0, 0, 0}
// 		headers CocaineHeaders
// 	)
// 	codec.NewDecoderBytes(buff, hAsocket).MustDecode(&headers)
//
// 	assert.Equal(t, 3, len(headers))
// 	for i, header := range headers {
// 		switch i {
// 		case 0:
// 			n, b, err := getTrace(header)
// 			assert.NoError(t, err)
// 			assert.Equal(t, uint64(traceId), n)
//
// 			trace, err := decodeTracingId(b)
// 			assert.NoError(t, err)
// 			assert.Equal(t, uint64(9000), trace)
// 		case 1:
// 			n, b, err := getTrace(header)
// 			assert.NoError(t, err)
// 			assert.Equal(t, uint64(spanId), n)
//
// 			span, err := decodeTracingId(b)
// 			assert.NoError(t, err)
// 			assert.Equal(t, uint64(11000), span)
// 		case 2:
// 			n, b, err := getTrace(header)
// 			assert.NoError(t, err)
// 			assert.Equal(t, uint64(parentId), n)
//
// 			parent, err := decodeTracingId(b)
// 			assert.NoError(t, err)
// 			assert.Equal(t, uint64(8000), parent)
// 		}
// 	}
//
// 	traceInfo, err := headers.getTraceData()
// 	assert.NoError(t, err)
// 	assert.Equal(t, uint64(9000), traceInfo.trace)
// 	assert.Equal(t, uint64(11000), traceInfo.span)
// 	assert.Equal(t, uint64(8000), traceInfo.parent)
// }
//
// func TestHeadersPackUnpack(t *testing.T) {
// 	trace := TraceInfo{
// 		trace:  uint64(100),
// 		span:   uint64(200),
// 		parent: uint64(300),
// 	}
//
// 	headers, err := traceInfoToHeaders(&trace)
// 	assert.NoError(t, err)
// 	assert.Equal(t, 3, len(headers))
// 	// t.Logf("%v", headers)
//
// 	traceInfo, err := headers.getTraceData()
// 	if !assert.NoError(t, err) {
// 		t.Logf("%v", traceInfo)
// 		t.FailNow()
// 	}
//
// 	assert.Equal(t, trace.trace, traceInfo.trace)
// 	assert.Equal(t, trace.span, traceInfo.span)
// 	assert.Equal(t, trace.parent, traceInfo.parent)
// }
//
// func BenchmarkTraceExtract(b *testing.B) {
// 	var (
// 		//trace.pack_trace(trace.Trace(traceid=9000, spanid=11000, parentid=8000))
// 		buff = []byte{
// 			147, 147, 194, 80, 168, 40, 35, 0, 0, 0, 0, 0, 0, 147, 194, 81, 168,
// 			248, 42, 0, 0, 0, 0, 0, 0, 147, 194, 82, 168, 64, 31, 0, 0, 0, 0, 0, 0}
// 		headers CocaineHeaders
// 	)
// 	codec.NewDecoderBytes(buff, hAsocket).MustDecode(&headers)
// 	b.ResetTimer()
// 	for n := 0; n < b.N; n++ {
// 		headers.getTraceData()
// 	}
// }
