package launcher

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/jenkins-x/jx/pkg/jxfactory"
	"github.com/jenkins-x/jx/pkg/tekton/metapipeline"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/jenkins-x/lighthouse/pkg/clients"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// NewMetaPipelineClient creates a new client for the creation and application of meta pipelines.
// The responsibility of the meta pipeline is to prepare the execution pipeline and to allow Apps to contribute
// the this execution pipeline.
func NewMetaPipelineClient(factory jxfactory.Factory) (metapipeline.Client, error) {
	if factory == nil {
		logrus.Warnf("no jxfactory passed in to create metapipeline.Client: %s", string(debug.Stack()))
		factory = jxfactory.NewFactory()
	}
	logrus.Info("creating a metapipeline client")
	// lets make sure that we have the jx home dir created
	cfgHome := util.HomeDir()
	err := os.MkdirAll(cfgHome, util.DefaultWritePermissions)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create jx home dir %s", cfgHome)
	}

	tektonClient, jxClient, kubeClient, _, ns, err := clients.GetClientsAndNamespace(factory)
	if err != nil {
		return nil, err
	}
	/*
		gitter := gits.NewGitCLI()
		fileHandles := util.IOFileHandles{
			Err: os.Stderr,
			In:  os.Stdin,
			Out: os.Stdout,
		}
	*/
	client, err := metapipeline.NewMetaPipelineClientWithClientsAndNamespace(jxClient, tektonClient, kubeClient, ns)
	if err == nil && client == nil {
		return nil, fmt.Errorf("no metapipeline client created")
	}
	return client, err
}
