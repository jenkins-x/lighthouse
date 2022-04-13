package pollstate

import "sync"

type memoryPollstate struct {
	cache map[string]string
	lock  sync.RWMutex
}

// NewMemoryPollState creates a new poll state
func NewMemoryPollState() Interface {
	return &memoryPollstate{
		cache: map[string]string{},
	}
}

func (p *memoryPollstate) IsNew(repository, operation, newValue string) (bool, error) {
	key := repository + ":" + operation

	p.lock.Lock()

	old := p.cache[key]
	p.cache[key] = newValue

	p.lock.Unlock()
	return old != newValue, nil
}

func (p *memoryPollstate) Invalidate(repository, operation, invalidValue string) {
	key := repository + ":" + operation

	p.lock.Lock()

	currentValue := p.cache[key]
	if currentValue == invalidValue {
		delete(p.cache, key)
	}

	p.lock.Unlock()
}
