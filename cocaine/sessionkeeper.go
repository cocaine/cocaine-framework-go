package cocaine

import (
	"sync"
)

type keeperStruct struct {
	sync.RWMutex
	links   map[int64](chan ServiceResult)
	counter int64
}

func newKeeperStruct() *keeperStruct {
	return &keeperStruct{links: make(map[int64]chan ServiceResult)}
}

func (keeper *keeperStruct) Attach(out chan ServiceResult) int64 {
	defer keeper.Unlock()
	keeper.Lock()
	keeper.counter++
	keeper.links[keeper.counter] = out
	return keeper.counter
}

func (keeper *keeperStruct) Detach(id int64) {
	defer keeper.Unlock()
	keeper.Lock()
	delete(keeper.links, id)
}

func (keeper *keeperStruct) Get(id int64) chan ServiceResult {
	defer keeper.RUnlock()
	keeper.RLock()
	i := keeper.links[id]
	return i
}
