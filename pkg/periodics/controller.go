package periodics

import (
	"time"

	lighthousev1alpha1 "github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned/typed/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	controllerName = "periodics-controller"
)

type LighthousePeriodicJobController struct {
	logger              *logrus.Entry
	queue               workqueue.RateLimitingInterface
	lighthouseJobClient lighthousev1alpha1.LighthouseJobInterface
	configAgent         *config.Agent
}

func NewLighthousePeriodicJobController(queue workqueue.RateLimitingInterface, lighthouseJobClient lighthousev1alpha1.LighthouseJobInterface, configAgent *config.Agent) *LighthousePeriodicJobController {
	return &LighthousePeriodicJobController{
		logger:              logrus.NewEntry(logrus.StandardLogger()).WithField("controller", controllerName),
		queue:               queue,
		lighthouseJobClient: lighthouseJobClient,
		configAgent:         configAgent,
	}
}

func (c *LighthousePeriodicJobController) Run(workerCount int, stopCh chan struct{}) {
	c.logger.Info("Starting controller")
	defer c.queue.ShutDown()

	for i := 0; i < workerCount; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	c.logger.Info("Stopping controller")
}

func (c *LighthousePeriodicJobController) runWorker() {
	for c.processNextItem() {
	}
}

// processNextItem takes items from the queue and reconciles them
func (c *LighthousePeriodicJobController) processNextItem() bool {
	// Wait until there is a new item in the working queue
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	// Tell the queue that we are done with processing this key. This unblocks
	// the key for other workers. This allows safe parallel processing because
	// the same key is never processed in parallel
	defer c.queue.Done(key)

	// Invoke the method containing the business logic
	err := c.reconcile(key.(ctrl.Request))

	// Handle the error if something went wrong with reconciliation
	c.handleErr(err, key)
	return true
}

// handleErr checks if an error happened and makes sure to retry later
func (c *LighthousePeriodicJobController) handleErr(err error, key interface{}) {
	if err == nil {
		// Forget key on successful reconciliation
		c.queue.Forget(key)
		return
	}

	// Retry with backoff if there was a reconciliation error
	c.logger.WithError(err).Infof("Failed to reconcile periodic job %s", key)
	c.queue.AddRateLimited(key)
}

// reconcile contains the business logic of the controller
func (c *LighthousePeriodicJobController) reconcile(req ctrl.Request) error {
	c.logger.Info("Reconciling periodic job %s...", req)

	// Retrieve the latest Lighthouse config and search for the definition of
	// the periodic job being reconciled
	for _, periodic := range c.configAgent.Config().JobConfig.Periodics {
		if periodic.Name == req.Name && periodic.Namespace != nil && *periodic.Namespace == req.Namespace {
			c.logger.Info("Periodic job configuration found!")
		} else {
			c.logger.Errorf("Failed to find configuration for periodic job %s", req)
			return nil
		}
	}

	// TODO: Finish the controller implementation by scheduling LighthouseJobs

	c.logger.Infof("Periodic job %s reconciled successfully!", req)
	return nil
}
