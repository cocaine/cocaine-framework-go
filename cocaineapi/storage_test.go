package cocaineapi

import (
	"os"
	"testing"
)

func init() {
	endpoint = os.Getenv(COCAINE_ENPOINT)
	if len(endpoint) == 0 {
		endpoint = "localhost:10053"
	}
}

func initializeStorage(t *testing.T) Storage {
	s, err := NewStorage(endpoint)
	if err != nil {
		t.Fatalf("Unexpected creation error %s", err)
	}
	return s
}

func TestStorageFind(t *testing.T) {
	s := initializeStorage(t)
	tags := []string{"manifests"}
	m, err := s.Find("manifest", tags)
	if err != nil {
		t.Fatalf("Error %s ", err)
	}
	t.Logf("Manifests: %s", m)
}

func TestStorageRead(t *testing.T) {
	s := initializeStorage(t)
	namespace := "apps"
	key := "a"
	b, err := s.Read(namespace, key)
	t.Logf("%s", b)
}
