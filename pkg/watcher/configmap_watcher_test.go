package watcher

import (
	"testing"
	"time"

	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

// recordingCallback captures the latest value passed to a ConfigMapEntryCallback.
type recordingCallback struct {
	values chan string
}

func (c *recordingCallback) record(value string) {
	c.values <- value
}

// TestNewConfigMapWatcherSyncsAndInvokesCallback verifies the watcher's cache
// syncs (no "failed to wait for caches to sync") and that the initial ConfigMap
// is delivered to the callback. With the WatchListClient feature default-on (k8s
// 1.35+, client-go v0.36.2), this only passes because createWatcher forwards the
// reflector options and wraps the ListWatch so the fake client — which reports
// unsupported WatchList semantics — falls back to classic LIST+WATCH.
func TestNewConfigMapWatcherSyncsAndInvokesCallback(t *testing.T) {
	ns := "jx"
	kubeClient := kubefake.NewSimpleClientset(&v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.ProwConfigMapName,
			Namespace: ns,
		},
		Data: map[string]string{
			util.ProwConfigFilename: "initial-config",
		},
	})

	rec := &recordingCallback{values: make(chan string, 1)}
	callbacks := []ConfigMapCallback{
		&ConfigMapEntryCallback{
			Name:     util.ProwConfigMapName,
			Key:      util.ProwConfigFilename,
			Callback: rec.record,
		},
	}

	stopCh := make(chan struct{})
	defer close(stopCh)

	w, err := NewConfigMapWatcher(kubeClient, ns, callbacks, stopCh)
	require.NoError(t, err, "watcher should start without a cache-sync failure")
	require.NotNil(t, w)
	defer w.Stop()

	select {
	case value := <-rec.values:
		assert.Equal(t, "initial-config", value)
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for ConfigMap callback to fire")
	}
}
