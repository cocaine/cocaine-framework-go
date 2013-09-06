package cocaine

import (
	"sync"
)

type Keeper struct {
	sync.RWMutex
	links   map[int64](chan ServiceResult)
	counter int64
}

func NewKeeper() *Keeper {
	return &Keeper{links: make(map[int64]chan ServiceResult)}
}

func (keeper *Keeper) Attach(out chan ServiceResult) int64 {
	defer keeper.Unlock()
	keeper.Lock()
	keeper.counter++
	keeper.links[keeper.counter] = out
	return keeper.counter
}

func (keeper *Keeper) Detach(id int64) {
	defer keeper.Unlock()
	keeper.Lock()
	delete(keeper.links, id)
}

func (keeper *Keeper) Get(id int64) chan ServiceResult {
	defer keeper.RUnlock()
	keeper.RLock()
	i := keeper.links[id]
	return i
}
