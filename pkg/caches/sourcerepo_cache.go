package caches

import (
	jxv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	jxclient "github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/sirupsen/logrus"
)

// SourceRepositoryCache the cache
type SourceRepositoryCache struct {
	resources *ResourceCache
}

// NewSourceRepositoryCache creates a new cache
func NewSourceRepositoryCache(jxClient jxclient.Interface, ns string) (*SourceRepositoryCache, error) {
	resources, err := NewResourceCache(jxClient, ns, "sourcerepositories", &jxv1.SourceRepository{})
	if err != nil {
		return nil, err
	}
	return &SourceRepositoryCache{
		resources,
	}, nil
}

// IsLoaded returns true if the cache is loaded
func (c *SourceRepositoryCache) IsLoaded() bool {
	return c.resources.IsLoaded()
}

// Stop closes the underlying chanel processing events which stops consuming watch events
func (c *SourceRepositoryCache) Stop() {
	c.resources.Stop()
}

// Get looks up the repository by name
func (c *SourceRepositoryCache) Get(name string) *jxv1.SourceRepository {
	answer := c.resources.Get(name)
	if answer != nil {
		sr, ok := answer.(*jxv1.SourceRepository)
		if ok {
			return sr
		}
	}
	return nil
}

// FindRepository looks up the repository by name
func (c *SourceRepositoryCache) FindRepository(owner string, name string) *jxv1.SourceRepository {
	var answer *jxv1.SourceRepository
	c.resources.Range(func(_, obj interface{}) bool {
		if answer != nil {
			return false
		}
		sr, ok := obj.(*jxv1.SourceRepository)
		if ok {
			if sr.Spec.Org == owner && sr.Spec.Repo == name {
				answer = sr
				return false
			}
		} else {
			logrus.Warnf("unknown object %#v in cache", obj)
		}
		return true
	})
	return answer
}
