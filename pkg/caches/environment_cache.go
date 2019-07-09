package caches

import (
	jxv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	jxclient "github.com/jenkins-x/jx/pkg/client/clientset/versioned"
)

// EnvironmentCache the cache
type EnvironmentCache struct {
	resources *ResourceCache
}

// NewEnvironmentCache creates a new cache
func NewEnvironmentCache(jxClient jxclient.Interface, ns string) (*EnvironmentCache, error) {
	resources, err := NewResourceCache(jxClient, ns, "environments", &jxv1.Environment{})
	if err != nil {
		return nil, err
	}
	return &EnvironmentCache{
		resources,
	}, nil
}

// Stop closes the underlying chanel processing events which stops consuming watch events
func (c *EnvironmentCache) Stop() {
	c.resources.Stop()
}

// IsLoaded returns true if the cache is loaded
func (c *EnvironmentCache) IsLoaded() bool {
	return c.resources.IsLoaded()
}

// Get looks up the repository by name
func (c *EnvironmentCache) Get(name string) *jxv1.Environment {
	answer := c.resources.Get(name)
	if answer != nil {
		sr, ok := answer.(*jxv1.Environment)
		if ok {
			return sr
		}
	}
	return nil
}
