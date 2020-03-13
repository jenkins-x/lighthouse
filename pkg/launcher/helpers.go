package launcher

import (
	"fmt"
	"os"
	"runtime/debug"

	jxclient "github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/jxfactory"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/tekton/metapipeline"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	kubeclient "k8s.io/client-go/kubernetes"
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

	tektonClient, jxClient, kubeClient, ns, err := getClientsAndNamespace(factory)
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

func getClientsAndNamespace(factory jxfactory.Factory) (tektonclient.Interface, jxclient.Interface, kubeclient.Interface, string, error) {
	tektonClient, _, err := factory.CreateTektonClient()
	if err != nil {
		return nil, nil, nil, "", errors.Wrap(err, "unable to create Tekton client")
	}

	jxClient, _, err := factory.CreateJXClient()
	if err != nil {
		return nil, nil, nil, "", errors.Wrap(err, "unable to create JX client")
	}

	kubeClient, ns, err := factory.CreateKubeClient()
	if err != nil {
		return nil, nil, nil, "", errors.Wrap(err, "unable to create Kube client")
	}
	ns, _, err = kube.GetDevNamespace(kubeClient, ns)
	if err != nil {
		return nil, nil, nil, "", errors.Wrap(err, "unable to find the dev namespace")
	}
	return tektonClient, jxClient, kubeClient, ns, nil
}
