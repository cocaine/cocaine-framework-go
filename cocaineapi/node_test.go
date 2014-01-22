package cocaineapi

import (
	"os"
	"testing"
)

const COCAINE_ENPOINT = "COCAINE"

var endpoint string

func init() {
	endpoint = os.Getenv(COCAINE_ENPOINT)
	if len(endpoint) == 0 {
		endpoint = "localhost:10053"
	}
}

func initializeNode(t *testing.T) Node {
	n, err := NewNode(endpoint)
	if err != nil {
		t.Fatalf("Unexpected creation error %s", err)
	}
	return n
}

func TestNodeAppList(t *testing.T) {
	n := initializeNode(t)
	l, err := n.AppList()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("App list %v", l)
}

func TestNodeAppStart(t *testing.T) {
	n := initializeNode(t)
	runlist := Runlist{"a": "a", "b": "b"}
	l, err := n.StartApp(runlist)
	if err != nil {
		t.Fatalf("Error %s", err)
	}
	t.Logf("%s", l)
}

func TestNodeAppPause(t *testing.T) {
	n := initializeNode(t)
	apps := []string{"a", "b"}
	l, err := n.PauseApp(apps)
	if err != nil {
		t.Fatalf("Error %s", err)
	}
	t.Logf("%s", l)
}
