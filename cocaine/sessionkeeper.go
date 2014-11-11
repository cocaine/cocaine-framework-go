package cocaine

import (
	"sync"
)

type keeperStruct struct {
	sync.RWMutex
	links   map[uint64](chan ServiceResult)
	counter uint64
}

func newKeeperStruct() *keeperStruct {
	return &keeperStruct{links: make(map[uint64]chan ServiceResult)}
}

func (keeper *keeperStruct) Attach(out chan ServiceResult) uint64 {
	keeper.Lock()
	defer keeper.Unlock()
	keeper.counter++
	keeper.links[keeper.counter] = out
	return keeper.counter
}

func (keeper *keeperStruct) Detach(id uint64) {
	keeper.Lock()
	defer keeper.Unlock()
	delete(keeper.links, id)
}

func (keeper *keeperStruct) Get(id uint64) (ch chan ServiceResult, ok bool) {
	keeper.RLock()
	defer keeper.RUnlock()
	ch, ok = keeper.links[id]
	return
}

func (keeper *keeperStruct) Keys() (keys []uint64) {
	keeper.RLock()
	defer keeper.RUnlock()
	for k := range keeper.links {
		keys = append(keys, k)
	}
	return
}
