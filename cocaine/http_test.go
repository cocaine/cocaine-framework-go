package cocaine

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"testing"

	"net/http"
)

func TestUnGzipBody(t *testing.T) {
	buffer := bytes.NewBuffer(nil)
	buffer.Write([]byte("AAAAAA"))
	req, err := http.NewRequest("POST", "/test", buffer)
	if err != nil {
		t.Fatalf("%v", err)
	}
	req.Header.Add("Content-Encoding", "gzip")
	if err := decompressBody(req); err == nil {
		t.Fatal("decompressBody: expect an error")
	}

	mock := []byte("ABCDEFGHJKLMNI")
	gzipBuffer := bytes.NewBuffer(nil)
	gzipWriter := gzip.NewWriter(gzipBuffer)
	gzipWriter.Write(mock)
	gzipWriter.Close()

	req, err = http.NewRequest("POST", "/test", gzipBuffer)
	if err != nil {
		t.Fatalf("%v", err)
	}
	req.Header.Add("Content-Encoding", "gzip")
	if err := decompressBody(req); err != nil {
		t.Fatalf("decompressBody %v", err)
	}
	defer req.Body.Close()

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("%v", err)
	}

	if !bytes.Equal(body, mock) {
		t.Fatalf("%s %s", body, mock)
	}
}
