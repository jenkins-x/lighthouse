package main

import (
	"time"

	"golang.org/x/time/rate"

	webhookset "github.com/foghornci/foghorn/pkg/client/clientset/versioned"
	"github.com/foghornci/foghorn/pkg/client/informers/externalversions"
	informersv1 "github.com/foghornci/foghorn/pkg/client/informers/externalversions/foghornci.io/v1"
	listersv1 "github.com/foghornci/foghorn/pkg/client/listers/foghornci.io/v1"
	"github.com/sirupsen/logrus"
	untypedcorev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

type controller struct {
	webhookLister   listersv1.WebhookLister
	webhookInformer cache.SharedIndexInformer
	workqueue       workqueue.RateLimitingInterface
	recorder        record.EventRecorder
}

const controllerName = "webhook-crd"

func newController(webhooks informersv1.WebhookInformer) *controller {
	kubeConfig, err := rest.InClusterConfig()
	if err != nil {
		logrus.WithError(err).Fatalf("failed to load k8s config")
	}

	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		logrus.WithError(err).Fatalf("failed to create k8s client")
	}

	webhookClient, err := webhookset.NewForConfig(kubeConfig)
	if err != nil {
		logrus.WithError(err).Fatalf("failed to create k8s client")
	}

	webhookInformerFactory := externalversions.NewSharedInformerFactory(webhookClient, 30*time.Minute)

	rl := workqueue.NewMaxOfRateLimiter(
		workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 120*time.Second),
		&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(1000), 50000)},
	)

	// Log to events
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(logrus.Infof)
	eventBroadcaster.StartRecordingToSink(&corev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})

	c := &controller{
		webhookLister:   webhookInformerFactory.Foghornci().V1().Webhooks().Lister(),
		webhookInformer: webhookInformerFactory.Foghornci().V1().Webhooks().Informer(),
		workqueue:       workqueue.NewNamedRateLimitingQueue(rl, controllerName),
		recorder:        eventBroadcaster.NewRecorder(scheme.Scheme, untypedcorev1.EventSource{Component: controllerName}),
	}

	webhookInformerFactory.Foghornci().V1().Webhooks().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				c.workqueue.Add(key)
			}
		},
		UpdateFunc: func(old, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				c.workqueue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				c.workqueue.Add(key)
			}
		},
	})
	return c
}
