package filebrowser

import "sync"

// FetchCache whether or not we should fetch the given repo ref
// we only need to fetch a given repo ref once per webhook request
type FetchCache interface {
	// ShouldFetch returns true if we should fetch for the given owner, repo and ref
	ShouldFetch(fullName, ref string) bool
}

// NewFetchCache creates a default fetch cache
func NewFetchCache() FetchCache {
	return &defaultFetchCache{fetched: map[string]bool{}}
}

type defaultFetchCache struct {
	lock    sync.RWMutex
	fetched map[string]bool
}

// ShouldFetch returns true if we should fetch for the given owner, repo and ref
func (f *defaultFetchCache) ShouldFetch(fullName, ref string) bool {
	key := fullName + "@" + ref

	f.lock.Lock()

	answer := false
	if !f.fetched[key] {
		answer = true
		f.fetched[key] = true
	}
	f.lock.Unlock()
	return answer
}
