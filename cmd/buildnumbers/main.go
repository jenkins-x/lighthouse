package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/jenkins-x/lighthouse/pkg/clients"
	"github.com/jenkins-x/lighthouse/pkg/interrupts"
	"github.com/jenkins-x/lighthouse/pkg/logrusutil"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclient "k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	configMapName = "lighthouse-build-numbers"
	configMapKey  = "buildNums.json"
)

type options struct {
	port      int
	namespace string
}

func gatherOptions() options {
	o := options{}
	flag.IntVar(&o.port, "port", 8888, "Port to listen on.")
	flag.StringVar(&o.namespace, "namespace", "", "The namespace to find the configmap in")

	flag.Parse()
	return o
}

func (o *options) Validate() error {
	return nil
}

type store struct {
	Number          map[string]int // job name -> last vended build number
	mutex           sync.Mutex
	configMapClient corev1.ConfigMapInterface
}

func newStore(configMapClient corev1.ConfigMapInterface) (*store, error) {
	s := &store{
		Number:          make(map[string]int),
		configMapClient: configMapClient,
	}
	cfgMap, err := configMapClient.Get(configMapName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch current state of configmap: %v", err)
	}
	value := cfgMap.Data[configMapKey]

	err = json.Unmarshal([]byte(value), s)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *store) save() error {
	buf, err := json.Marshal(s)
	if err != nil {
		return err
	}
	cfgMap, err := s.configMapClient.Get(configMapName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to fetch current state of configmap: %v", err)
	}

	cfgMap.Data[configMapKey] = string(buf)
	_, err = s.configMapClient.Update(cfgMap)
	return err
}

func (s *store) vend(jobName string) int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	n, ok := s.Number[jobName]
	if !ok {
		n = 0
	}
	n++

	s.Number[jobName] = n

	err := s.save()
	if err != nil {
		logrus.Error(err)
	}

	return n
}

func (s *store) peek(jobName string) int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.Number[jobName]
}

func (s *store) set(jobName string, n int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.Number[jobName] = n

	err := s.save()
	if err != nil {
		logrus.Error(err)
	}
}

func (s *store) handle(w http.ResponseWriter, r *http.Request) {
	jobName := r.URL.Path[len("/vend/"):]
	switch r.Method {
	case "GET":
		n := s.vend(jobName)
		logrus.Infof("Vending %s number %d to %s.", jobName, n, r.RemoteAddr)
		fmt.Fprintf(w, "%d", n)
	case "HEAD":
		n := s.peek(jobName)
		logrus.Infof("Peeking %s number %d to %s.", jobName, n, r.RemoteAddr)
		fmt.Fprintf(w, "%d", n)
	case "POST":
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			logrus.WithError(err).Error("Unable to read body.")
			return
		}
		n, err := strconv.Atoi(string(body))
		if err != nil {
			logrus.WithError(err).Error("Unable to parse number.")
			return
		}
		logrus.Infof("Setting %s to %d from %s.", jobName, n, r.RemoteAddr)
		s.set(jobName, n)
	}
}

func main() {
	logrusutil.ComponentInit("lighthouse-build-numbers")

	o := gatherOptions()
	if err := o.Validate(); err != nil {
		logrus.Fatalf("Invalid options: %v", err)
	}

	defer interrupts.WaitForGracefulShutdown()

	kubeCfg, err := clients.GetConfig("", "")
	if err != nil {
		logrus.WithError(err).Fatal("Could not create kubeconfig")
	}

	kubeClient, err := kubeclient.NewForConfig(kubeCfg)
	if err != nil {
		logrus.WithError(err).Fatal("Could not create kubeclient")
	}

	health := util.NewHealth()

	s, err := newStore(kubeClient.CoreV1().ConfigMaps(o.namespace))
	if err != nil {
		logrus.WithError(err).Fatal("newStore failed")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/vend/", s.handle)
	server := &http.Server{Addr: ":" + strconv.Itoa(o.port), Handler: mux}
	health.ServeReady()
	interrupts.ListenAndServe(server, 5*time.Second)
}
