package cocaine12

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestReaderEOF(t *testing.T) {
	type tStruct struct {
		L string
		N int
	}

	ctx := context.Background()

	chunks := []tStruct{
		{"A", 100},
		{"B", 101},
		{"C", 102},
		{"D", 102},
	}

	req := newRequest(newV1Protocol())
	for _, m := range chunks {
		body, _ := json.Marshal(m)
		req.push(newChunkV1(2, body))
	}
	req.Close()

	var actual tStruct
	dec := json.NewDecoder(RequestReader(ctx, req))
	for i := range chunks {
		err := dec.Decode(&actual)
		assert.NoError(t, err)
		assert.Equal(t, chunks[i], actual)
	}

	err := dec.Decode(&actual)
	assert.EqualError(t, err, io.EOF.Error())
}

func TestRequestReaderErrorV1(t *testing.T) {
	type tStruct struct {
		L string
		N int
	}

	ctx := context.Background()

	chunks := []tStruct{
		{"A", 100},
		{"B", 101},
		{"C", 102},
		{"D", 102},
	}

	req := newRequest(newV1Protocol())
	for _, m := range chunks {
		body, _ := json.Marshal(m)
		req.push(newChunkV1(2, body))
	}
	req.push(newErrorV1(2, 100, 200, "error"))

	var actual tStruct
	dec := json.NewDecoder(RequestReader(ctx, req))
	for i := range chunks {
		err := dec.Decode(&actual)
		assert.NoError(t, err)
		assert.Equal(t, chunks[i], actual)
	}

	err := dec.Decode(&actual)
	expectedV1 := &ErrRequest{"error", 100, 200}
	assert.EqualError(t, err, expectedV1.Error())
}

func TestServiceResult(t *testing.T) {
	sr := serviceRes{
		payload: []interface{}{"A", 100},
		err:     nil,
	}

	var (
		s string
		i int
	)
	err := sr.ExtractTuple(&s, &i)
	assert.NoError(t, err)
	assert.Equal(t, "A", s)
	assert.Equal(t, 100, i)
}
