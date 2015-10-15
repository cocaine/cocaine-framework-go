package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMain(t *testing.T) {
	mux := NewServer()
	ts := httptest.NewServer(mux)
	defer ts.Close()

	res, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, http.StatusBadRequest, res.StatusCode)

	res, err = http.Get(ts.URL + "/A")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, http.StatusBadRequest, res.StatusCode)

	res, err = http.Get(ts.URL + "/A/B")
	if err != nil {
		t.Fatal(err)
	}

	res, err = http.Get(ts.URL + "/A/B/URL")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, http.StatusOK, res.StatusCode)
}
