package cocaine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocatorResolve(t *testing.T) {
	const (
		name = "node"
	)

	l, err := NewLocator()
	if err != nil {
		t.Fatalf("unable to create locator %s", err)
	}
	defer l.Close()

	r := <-l.Resolve(name)
	if r.Err != nil {
		t.Fatalf("unable to unpack locator API %s", r.Err)
	}

	assert.Equal(t, r.Version, 1, "Wrong API version")
	for _, k := range r.Endpoints {
		t.Log(k.String())
	}

	if i, err := r.API.MethodByName("start_app"); err != nil {
		t.Fatal(err)
	} else if !assert.Equal(t, 0, i) {
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

	r := <-n.Call("list")
	t.Log(r)

	r = <-n.Call("start_app", "A", "default")
	t.Log(r)

}

func TestStorage(t *testing.T) {
	s, err := NewService("storage")
	if err != nil {
		t.Fatalf("unable to crete node service %s", err)
	}
	r := <-s.Call("find", "profiles", []string{"profile"})
	t.Log(r)
}
