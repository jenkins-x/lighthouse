package main

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	ns = "test-namespace"
)

func expectEqual(t *testing.T, msg string, have interface{}, want interface{}) {
	if !reflect.DeepEqual(have, want) {
		t.Errorf("bad %s: got %v, wanted %v",
			msg, have, want)
	}
}

func makeStore(t *testing.T) *store {
	startingConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: ns,
		},
		Data: map[string]string{
			configMapKey: "{ \"Number\": {} }",
		},
	}

	fkc := fake.NewSimpleClientset(startingConfigMap)

	cfgMapClient := fkc.CoreV1().ConfigMaps(ns)
	store, err := newStore(cfgMapClient)
	if err != nil {
		t.Fatal(err)
	}

	return store
}

func TestVend(t *testing.T) {
	store := makeStore(t)

	expectEqual(t, "empty vend", store.vend("a"), 1)
	expectEqual(t, "second vend", store.vend("a"), 2)
	expectEqual(t, "third vend", store.vend("a"), 3)
	expectEqual(t, "second empty", store.vend("b"), 1)

	store2, err := newStore(store.configMapClient)
	assert.NoError(t, err)
	expectEqual(t, "fourth vend, different instance", store2.vend("a"), 4)
}

func TestSet(t *testing.T) {
	store := makeStore(t)

	store.set("foo", 300)
	expectEqual(t, "peek", store.peek("foo"), 300)
	store.set("foo2", 300)
	expectEqual(t, "vend", store.vend("foo2"), 301)
	expectEqual(t, "vend", store.vend("foo2"), 302)
}

func expectResponse(t *testing.T, handler http.Handler, req *http.Request, msg, value string) {
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	expectEqual(t, "http status OK", rr.Code, 200)

	expectEqual(t, msg, rr.Body.String(), value)
}

func TestHandler(t *testing.T) {
	store := makeStore(t)

	handler := http.HandlerFunc(store.handle)

	req, err := http.NewRequest("GET", "/vend/foo", nil)
	if err != nil {
		t.Fatal(err)
	}

	expectResponse(t, handler, req, "http vend", "1")
	expectResponse(t, handler, req, "http vend", "2")

	req, err = http.NewRequest("HEAD", "/vend/foo", nil)
	if err != nil {
		t.Fatal(err)
	}

	expectResponse(t, handler, req, "http peek", "2")

	req, err = http.NewRequest("POST", "/vend/bar", strings.NewReader("40"))
	if err != nil {
		t.Fatal(err)
	}

	expectResponse(t, handler, req, "http post", "")

	req, err = http.NewRequest("HEAD", "/vend/bar", nil)
	if err != nil {
		t.Fatal(err)
	}

	expectResponse(t, handler, req, "http vend", "40")
}
