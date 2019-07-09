package caches

import (
	jxv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	jxclient "github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/sirupsen/logrus"
)

// SchedulerCache the cache
type SchedulerCache struct {
	resources *ResourceCache
}

// NewSchedulerCache creates a new cache
func NewSchedulerCache(jxClient jxclient.Interface, ns string) (*SchedulerCache, error) {
	resources, err := NewResourceCache(jxClient, ns, "schedulers", &jxv1.Scheduler{})
	if err != nil {
		return nil, err
	}
	return &SchedulerCache{
		resources,
	}, nil
}

// Stop closes the underlying chanel processing events which stops consuming watch events
func (c *SchedulerCache) Stop() {
	c.resources.Stop()
}

// IsLoaded returns true if the cache is loaded
func (c *SchedulerCache) IsLoaded() bool {
	return c.resources.IsLoaded()
}

// Get looks up the repository by name
func (c *SchedulerCache) Get(name string) *jxv1.Scheduler {
	answer := c.resources.Get(name)
	if answer != nil {
		sr, ok := answer.(*jxv1.Scheduler)
		if ok {
			return sr
		}
	}
	return nil
}

// List finds all the schedulers
func (c *SchedulerCache) List(owner string, name string) []*jxv1.Scheduler {
	answer := []*jxv1.Scheduler{}
	c.resources.Range(func(_, obj interface{}) bool {
		if answer != nil {
			return false
		}
		r, ok := obj.(*jxv1.Scheduler)
		if ok {
			answer = append(answer, r)
		} else {
			logrus.Warnf("unknown object %#v in cache", obj)
		}
		return true
	})
	return answer
}
