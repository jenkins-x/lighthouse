package caches

import "time"

// WaitForCachesToLoad waits for all the caches to be loaded
func WaitForCachesToLoad(caches ...Cache) {
	for {
		loaded := true

		for _, cache := range caches {
			if !cache.IsLoaded() {
				loaded = false
			}
		}

		if loaded {
			return
		}

		time.Sleep(time.Second)
	}
}
