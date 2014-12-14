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
	defer keeper.Unlock()
	keeper.counter++
	keeper.links[keeper.counter] = rx
	return keeper.counter
}

func (keeper *keeperStruct) Detach(id uint64) {
	keeper.Lock()
	defer keeper.Unlock()
	delete(keeper.links, id)
}

func (keeper *keeperStruct) Get(id uint64) (Rx, bool) {
	keeper.RLock()
	defer keeper.RUnlock()
	rx, ok := keeper.links[id]
	return rx, ok
}

func (keeper *keeperStruct) Keys() (keys []uint64) {
	keeper.RLock()
	defer keeper.RUnlock()
	for k := range keeper.links {
		keys = append(keys, k)
	}
	return
}
