package cocaineapi

import (
	"github.com/cocaine/cocaine-framework-go/cocaine"
)

type Storage interface {
	//Read(namespace string, key string) ([]byte, error)
	Find(namespace string, tags []string) ([]string, error)
}

type storage struct {
	s *cocaine.Service
}

func NewStorage(endpoint string) (st Storage, err error) {
	s, err := cocaine.NewService("storage", endpoint)
	if err != nil {
		return
	}
	st = &storage{s}
	return
}

func (s *storage) Find(namespace string, tags []string) (keys []string, err error) {
	res := <-s.s.Call("find", namespace, tags)
	err = res.Err()
	if err != nil {
		return
	}
	res.Extract(&keys)
	return
}
