package caches

import (
	"sync"
	"time"

	jxclient "github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

// Cache handles interactions with the cache
type Cache interface {
	// IsLoaded returns true when the cache is loaded
	IsLoaded() bool

	// Stop stops processing any updates to the cache as we are shutting down
	Stop()
}

// ResourceCache the cache
type ResourceCache struct {
	resources sync.Map
	stop      chan struct{}
	ready     bool
}

// NewResourceCache creates a new cache
func NewResourceCache(jxClient jxclient.Interface, ns string, resources string, resource runtime.Object) (*ResourceCache, error) {
	logrus.Debugf("caching the %s resources in namespace %s", resources, ns)

	listWatch := cache.NewListWatchFromClient(jxClient.JenkinsV1().RESTClient(), resources, ns, fields.Everything())

	resourceCache := &ResourceCache{
		stop: make(chan struct{}),
	}

	_, informer := cache.NewInformer(
		listWatch,
		resource,
		time.Minute*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				resourceCache.onResourceObj(obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				resourceCache.onResourceObj(newObj)
			},
			DeleteFunc: func(obj interface{}) {
				resourceCache.onPipelineDelete(obj)
			},
		},
	)

	go informer.Run(resourceCache.stop)
	return resourceCache, nil

}

func getName(obj interface{}) string {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		logrus.Warnf("failed to create Accessor for object  %#v", obj)
		return ""
	}
	return accessor.GetName()
}

func (c *ResourceCache) onResourceObj(obj interface{}) {
	name := getName(obj)
	if name != "" {
		c.resources.Store(name, obj)
	}
}

func (c *ResourceCache) onPipelineDelete(obj interface{}) {
	name := getName(obj)
	if name != "" {
		c.resources.Delete(name)
	}
}

// Stop closes the underlying chanel processing events which stops consuming watch events
func (c *ResourceCache) Stop() {
	c.ready = false
	close(c.stop)
}

// IsLoaded returns true if loaded
func (c *ResourceCache) IsLoaded() bool {
	// TODO there's no way to detect when all the List items have been procesesd yet
	// so for now lets just wait until we have at least one resource
	return c.Size() > 0
}

// Get looks up the object in the repository
func (c *ResourceCache) Get(name string) interface{} {
	answer, exists := c.resources.Load(name)
	if exists {
		return answer
	}
	return nil
}

// Size returns the number of cached objects
func (c *ResourceCache) Size() int {
	size := 0
	c.resources.Range(func(_, obj interface{}) bool {
		if obj != nil {
			size++
		}
		return true
	})
	return size
}

// Range objects returning false from the function to stop iterating
func (c *ResourceCache) Range(fn func(_, obj interface{}) bool) {
	c.resources.Range(fn)
}
