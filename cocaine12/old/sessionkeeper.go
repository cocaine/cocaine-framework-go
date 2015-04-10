package cocaine

import (
	"sync"
)

type keeperStruct struct {
	sync.RWMutex
	links   map[uint64]Rx
	counter uint64
}

func newKeeperStruct() *keeperStruct {
	return &keeperStruct{
		links:   make(map[uint64]Rx),
		counter: 1,
	}
}

func (keeper *keeperStruct) Attach(rx Rx) uint64 {
	keeper.Lock()

	keeper.counter++
	keeper.links[keeper.counter] = rx
	current := keeper.counter

	keeper.Unlock()
	return current
}

func (keeper *keeperStruct) Detach(id uint64) {
	keeper.Lock()

	delete(keeper.links, id)

	keeper.Unlock()
}

func (keeper *keeperStruct) Get(id uint64) (Rx, bool) {
	keeper.RLock()

	rx, ok := keeper.links[id]

	keeper.RUnlock()
	return rx, ok
}

func (keeper *keeperStruct) Keys() []uint64 {
	keeper.RLock()

	var keys = make([]uint64, len(keeper.links))
	for k := range keeper.links {
		keys = append(keys, k)
	}

	keeper.RUnlock()
	return keys
}
