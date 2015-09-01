package cocaine12

import (
	"fmt"
)

type dispatchType int

const (
	recursiveDispatch dispatchType = iota
	emptyDispatch
	otherDispatch
)

type dispatchMap map[uint64]dispatchItem

func (d *dispatchMap) Methods() []string {
	var methods = make([]string, 0, len(*d))
	for _, v := range *d {
		methods = append(methods, v.Name)
	}
	return methods
}

func (d *dispatchMap) MethodByName(name string) (uint64, error) {
	for i, v := range *d {
		if v.Name == name {
			return i, nil
		}
	}
	return 0, fmt.Errorf("no `%s` method", name)
}

type dispatchItem struct {
	Name       string
	Downstream *streamDescription
	Upstream   *streamDescription
}

type StreamDescriptionItem struct {
	Name        string
	Description *streamDescription
}

type streamDescription map[uint64]*StreamDescriptionItem

func (s *streamDescription) MethodByName(name string) (uint64, error) {
	for i, v := range *s {
		if v.Name == name {
			return i, nil
		}
	}

	return 0, fmt.Errorf("no `%s` method", name)
}

func (s *streamDescription) Type() dispatchType {
	switch {
	case s == nil:
		return recursiveDispatch
	case len(*s) == 0:
		return emptyDispatch
	default:
		return otherDispatch
	}
}
