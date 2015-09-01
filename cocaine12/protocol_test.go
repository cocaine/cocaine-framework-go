package cocaine12

import (
	"bytes"
	"sort"
	"testing"

	"github.com/cocaine/cocaine-framework-go/vendor/src/github.com/ugorji/go/codec"

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
