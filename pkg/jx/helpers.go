package jx

import (
	"fmt"
	"os"
	"runtime/debug"

	jxclient "github.com/jenkins-x/jx-api/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/v2/pkg/jxfactory"
	"github.com/jenkins-x/jx/v2/pkg/tekton/metapipeline"
	"github.com/jenkins-x/jx/v2/pkg/util"
	clientset "github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned"
	"github.com/jenkins-x/lighthouse/pkg/clients"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	kubeclient "k8s.io/client-go/kubernetes"
)

// NewMetaPipelineClient creates a new client for the creation and application of meta pipelines.
// The responsibility of the meta pipeline is to prepare the execution pipeline and to allow Apps to contribute
// the this execution pipeline.
func NewMetaPipelineClient(factory jxfactory.Factory) (metapipeline.Client, tektonclient.Interface, jxclient.Interface, kubeclient.Interface, clientset.Interface, string, error) {
	if factory == nil {
		logrus.Warnf("no jxfactory passed in to create metapipeline.Client: %s", string(debug.Stack()))
		factory = jxfactory.NewFactory()
	}
	logrus.Info("creating a metapipeline client")
	// lets make sure that we have the jx home dir created
	cfgHome := util.HomeDir()
	err := os.MkdirAll(cfgHome, util.DefaultWritePermissions)
	if err != nil {
		return nil, nil, nil, nil, nil, "", errors.Wrapf(err, "failed to create jx home dir %s", cfgHome)
	}

	tektonClient, jxClient, kubeClient, lhClient, ns, err := clients.GetClientsAndNamespace(factory)
	if err != nil {
		return nil, nil, nil, nil, nil, "", err
	}
	client, err := metapipeline.NewMetaPipelineClientWithClientsAndNamespace(jxClient, tektonClient, kubeClient, ns)
	if err == nil && client == nil {
		return nil, nil, nil, nil, nil, "", fmt.Errorf("no metapipeline client created")
	}
	return client, tektonClient, jxClient, kubeClient, lhClient, ns, err
}
