package watcher

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"
)

// ConfigMapWatcher callbacks for changes to a config map
type ConfigMapWatcher struct {
	kubeClient kubernetes.Interface
	namespace  string
	callbacks  []ConfigMapCallback
	watch      watch.Interface
	stopped    bool
	stopCh     <-chan struct{}
}

// ConfigMapCallback represents a callback
type ConfigMapCallback interface {
	OnChange(configMap *v1.ConfigMap)
}

// ConfigMapEntryCallback invokes a callback if the value changes
type ConfigMapEntryCallback struct {
	Name     string
	Key      string
	Callback func(string)
	oldValue string
}

// OnChange invokes the callback function if the value is not empty and changes
func (cb *ConfigMapEntryCallback) OnChange(configMap *v1.ConfigMap) {
	if cb.Name == configMap.Name {
		data := configMap.Data
		if data != nil {
			value := data[cb.Key]
			if value != "" {
				if value != cb.oldValue {
					cb.oldValue = value
					cb.Callback(value)
				}
			}
		}
	}
}

// NewConfigMapWatcher creates a new watcher of ConfigMap resources which lists them all synchronously then
// asynchronously processes watch events
func NewConfigMapWatcher(kubeClient kubernetes.Interface, ns string, callbacks []ConfigMapCallback, stopCh <-chan struct{}) (*ConfigMapWatcher, error) {
	w := &ConfigMapWatcher{
		kubeClient: kubeClient,
		namespace:  ns,
		// lets take a copy of the slice
		callbacks: append([]ConfigMapCallback{}, callbacks...),
		stopCh:    stopCh,
	}

	configMaps := kubeClient.CoreV1().ConfigMaps(ns)
	err := w.createWatcher()
	if err != nil {
		return w, err
	}

	// lets synchronously process the list resources, then async handle callbacks
	list, err := configMaps.List(metav1.ListOptions{})
	if err != nil {
		return w, errors.Wrapf(err, "failed to list ConfigMaps in namespace %s", ns)
	}

	for _, cm := range list.Items {
		w.invokeCallbacks(&cm)
	}

	// now lets asynchronously watch the events in the background
	go w.watchChannel()
	return w, nil
}

// IsStopped checks if configmap watcher is stopped
func (w *ConfigMapWatcher) IsStopped() bool {
	return w.stopped
}

// Stop stops the configmap watcher
func (w *ConfigMapWatcher) Stop() {
	w.stopped = true
	w.watch.Stop()
}

func (w *ConfigMapWatcher) invokeCallbacks(configMap *v1.ConfigMap) {
	for _, cb := range w.callbacks {
		cb.OnChange(configMap)
	}
}

func (w *ConfigMapWatcher) watchChannel() {
	l := logrus.WithField("namespace", w.namespace).WithField("component", "ConfigMapWatcher")

	for {
		for event := range w.watch.ResultChan() {
			switch event.Type {
			case watch.Added, watch.Modified:
				cm, ok := event.Object.(*v1.ConfigMap)
				if ok {
					w.invokeCallbacks(cm)
				} else {
					l.Errorf("unexpected event type: %#v", event.Object)
				}
			case watch.Error:
				if w.stopped {
					return
				}
				l.Errorf("failed with event %#v", event.Object)
			}
		}
	}
}

func (w *ConfigMapWatcher) createWatcher() error {
	if w.watch != nil {
		w.watch.Stop()
	}
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return w.kubeClient.CoreV1().ConfigMaps(w.namespace).List(metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return w.kubeClient.CoreV1().ConfigMaps(w.namespace).Watch(metav1.ListOptions{})
		},
	}
	var informer cache.Controller
	_, informer, w.watch = watchtools.NewIndexerInformerWatcher(lw, &v1.ConfigMap{})
	if ok := cache.WaitForCacheSync(w.stopCh, informer.HasSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	return nil
}
