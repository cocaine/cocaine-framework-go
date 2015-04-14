package cocaine12

import (
	"sync"
)

type sessions struct {
	sync.RWMutex
	links   map[uint64]Channel
	counter uint64
}

func newSessions() *sessions {
	return &sessions{
		links:   make(map[uint64]Channel),
		counter: 1,
	}
}

func (s *sessions) Attach(session Channel) uint64 {
	s.Lock()

	s.counter++
	s.links[s.counter] = session
	current := s.counter

	s.Unlock()
	return current
}

func (s *sessions) Detach(id uint64) {
	s.Lock()

	delete(s.links, id)

	s.Unlock()
}

func (s *sessions) Get(id uint64) (Channel, bool) {
	s.RLock()

	session, ok := s.links[id]

	s.RUnlock()
	return session, ok
}

func (s *sessions) Keys() []uint64 {
	s.RLock()

	var keys = make([]uint64, 0, len(s.links))
	for k := range s.links {
		keys = append(keys, k)
	}

	s.RUnlock()
	return keys
}
