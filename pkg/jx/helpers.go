package jx

import (
	"fmt"
	"os"

	jxclient "github.com/jenkins-x/jx-api/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/v2/pkg/tekton/metapipeline"
	"github.com/jenkins-x/jx/v2/pkg/util"
	clientset "github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned"
	"github.com/jenkins-x/lighthouse/pkg/clients"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// NewMetaPipelineClient creates a new client for the creation and application of meta pipelines.
// The responsibility of the meta pipeline is to prepare the execution pipeline and to allow Apps to contribute
// the this execution pipeline.
func NewMetaPipelineClient(ns string) (metapipeline.Client, jxclient.Interface, clientset.Interface, error) {
	logrus.Info("creating a metapipeline client")
	// lets make sure that we have the jx home dir created
	cfgHome := util.HomeDir()
	err := os.MkdirAll(cfgHome, util.DefaultWritePermissions)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed to create jx home dir %s", cfgHome)
	}

	tektonClient, kubeClient, lhClient, kubeCfg, err := clients.GetAPIClients()
	if err != nil {
		return nil, nil, nil, err
	}
	jxClient, err := jxclient.NewForConfig(kubeCfg)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "unable to create Jenkins X client")
	}

	client, err := metapipeline.NewMetaPipelineClientWithClientsAndNamespace(jxClient, tektonClient, kubeClient, ns)
	if err == nil && client == nil {
		return nil, nil, nil, fmt.Errorf("no metapipeline client created")
	}
	return client, jxClient, lhClient, err
}
