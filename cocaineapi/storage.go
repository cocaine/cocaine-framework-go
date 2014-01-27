package cocaineapi

import (
	"github.com/cocaine/cocaine-framework-go/cocaine"
)

type Storage interface {
	Find(namespace string, tags []string) ([]string, error)
	Read(namespace string, key string) (string, error)
	Write(namespace string, key string, blob string, tags []string) error
	Remove(namespace string, key string) error
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

func (s *storage) Read(namespace string, key string) (blob string, err error) {
	res := <-s.s.Call("read", namespace, key)
	err = res.Err()
	if err != nil {
		return
	}
	res.Extract(&blob)
	return
}

func (s *storage) Write(namespace string, key string, blob string, tags []string) (err error) {
	res, opened := <-s.s.Call("write", namespace, key, blob, tags)
	if opened {
		err = res.Err()
	}
	return
}

func (s *storage) Remove(namespace string, key string) (err error) {
	res, opened := <-s.s.Call("remove", namespace, key)
	if opened {
		err = res.Err()
	}
	return
}
