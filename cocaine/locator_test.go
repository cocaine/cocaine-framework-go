package cocaine

import (
	"log"
	"os"
	"testing"
)

func init() {
	if testing.Verbose() {
		l := log.New(os.Stderr, "[DEBUG TEST] ", log.LstdFlags)
		DEBUGTEST = l.Printf
	}
}

func TestLocatorResolve(t *testing.T) {
	const (
		name = "node"
	)

	l, err := NewLocator()
	if err != nil {
		t.Fatalf("unable to create locator %s", err)
	}
	defer l.Close()

	ch, err := l.Resolve(name)
	if err != nil {
		t.Fatal(err)
	}

	r := <-ch
	if r.Err != nil {
		t.Fatalf("unable to unpack locator API %s", r.Err)
	}

	if r.Version != 1 {
		t.Log("Wrong API version")
	}

	for _, k := range r.Endpoints {
		t.Log(k.String())
	}

	if i, err := r.API.MethodByName("start_app"); err != nil {
		t.Fatal(err)
	} else if i != 0 {
		t.Fatal("Unexpected method code", i)
	}

	if _, err := r.API.MethodByName("nonpresetmethod"); err == nil {
		t.Fatal("Error expected, got nil")
	}
}

func TestNode(t *testing.T) {
	n, err := NewService("node")
	if err != nil {
		t.Fatalf("unable to crete node service %s", err)
	}

	ch, err := n.Call("list")
	if err != nil {
		t.Fatal(err)
	}

	r, err := ch.Get()
	if err != nil {
		t.Fatal(err)
	}

	var listing struct {
		Data []string
	}
	if err := r.Extract(&listing); err != nil {
		t.Fatalf("unable to unpack node.list %v", err)
	}
	t.Log(listing.Data)
}

func TestStorage(t *testing.T) {
	s, err := NewService("storage")
	if err != nil {
		t.Fatalf("unable to crete node service %s", err)
	}
	ch, err := s.Call("find", "profiles", []string{"profile"})
	if err != nil {
		t.Fatalf("Unable to create channel for storage.Find: %s", err)
	}
	res, err := ch.Get()
	var listing struct {
		Data []string
	}
	if err := res.Extract(&listing); err != nil {
		t.Fatalf("unable to unpack storage.Fist %v", err)
	}
}
