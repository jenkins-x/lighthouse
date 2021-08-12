package watcher

import (
	"fmt"
	"sync"
	"time"

	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned"
	"github.com/sirupsen/logrus"

	informers "github.com/jenkins-x/lighthouse/pkg/client/informers/externalversions"
	"k8s.io/client-go/tools/cache"
)

// BreakpointWatcher lists and watches for breakpoints and caches them in memory
type BreakpointWatcher struct {
	lhClient  versioned.Interface
	namespace string
	logger    *logrus.Entry
	stopped   bool
	stop      chan struct{}
	lock      sync.RWMutex
	cache     map[string]*v1alpha1.LighthouseBreakpoint
}

// NewBreakpointWatcher creates a new watcher of Breakpoint resources which lists them all synchronously then
// asynchronously processes watch events
func NewBreakpointWatcher(lhClient versioned.Interface, ns string, logger *logrus.Entry) (*BreakpointWatcher, error) {
	if logger == nil {
		logger = logrus.NewEntry(logrus.StandardLogger()).WithField("controller", "BreakpointWatcher")
	}
	w := &BreakpointWatcher{
		lhClient:  lhClient,
		namespace: ns,
		logger:    logger,
		stop:      make(chan struct{}),
		cache:     map[string]*v1alpha1.LighthouseBreakpoint{},
	}

	informerFactory := informers.NewSharedInformerFactoryWithOptions(
		lhClient,
		time.Minute*10,
		informers.WithNamespace(ns),
	)

	informer := informerFactory.Lighthouse().V1alpha1().LighthouseBreakpoints().Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			e := obj.(*v1alpha1.LighthouseBreakpoint)
			if e != nil {
				w.onBreakpoint(e)
			} else {
				w.logger.Warnf("got invalid object %#v", obj)
			}
		},
		UpdateFunc: func(old, new interface{}) {
			e := new.(*v1alpha1.LighthouseBreakpoint)
			if e != nil {
				w.onBreakpoint(e)
			} else {
				w.logger.Warnf("got invalid object %#v", new)
			}
		},
		DeleteFunc: func(obj interface{}) {
			e := obj.(*v1alpha1.LighthouseBreakpoint)
			if e != nil {
				w.deleteBreakpoint(e.Name)
			} else {
				w.logger.Warnf("got invalid object %#v", obj)
			}
		},
	})
	informerFactory.Start(w.stop)

	if !cache.WaitForCacheSync(w.stop, informer.HasSynced) {
		msg := "timed out waiting for breakpoint caches to sync"
		return w, fmt.Errorf(msg)
	}

	count := len(w.GetBreakpoints())
	w.logger.Infof("on startup BreakpointWatcher has %d breakpoints", count)
	return w, nil
}

// IsStopped checks if the watcher is stopped
func (w *BreakpointWatcher) IsStopped() bool {
	return w.stopped
}

// Stop stops the watcher watcher
func (w *BreakpointWatcher) Stop() {
	if w.IsStopped() {
		return
	}
	w.stopped = true
	close(w.stop)
}

func (w *BreakpointWatcher) onBreakpoint(r *v1alpha1.LighthouseBreakpoint) {
	w.logger.Debugf("on Breakpoint: %s", r.Name)

	w.lock.Lock()

	w.cache[r.Name] = r

	w.lock.Unlock()
}

func (w *BreakpointWatcher) deleteBreakpoint(name string) {
	w.logger.Debugf("deleted Breakpoint: %s", name)
	w.lock.Lock()

	delete(w.cache, name)

	w.lock.Unlock()
}

// GetBreakpoints returns the current breakpoints
func (w *BreakpointWatcher) GetBreakpoints() []*v1alpha1.LighthouseBreakpoint {
	var answer []*v1alpha1.LighthouseBreakpoint

	w.lock.Lock()

	for _, r := range w.cache {
		answer = append(answer, r)
	}

	w.lock.Unlock()
	return answer
}
