package clients

import (
	clientset "github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned"
	"github.com/pkg/errors"

	jxclient "github.com/jenkins-x/jx-api/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/v2/pkg/jxfactory"
	"github.com/jenkins-x/jx/v2/pkg/kube"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	kubeclient "k8s.io/client-go/kubernetes"
)

// GetClientsAndNamespace returns the tekton, jx, kube, and Lighthouse clients and the dev namespace
func GetClientsAndNamespace(factory jxfactory.Factory) (tektonclient.Interface, jxclient.Interface, kubeclient.Interface, clientset.Interface, string, error) {
	if factory == nil {
		factory = jxfactory.NewFactory()
	}

	tektonClient, _, err := factory.CreateTektonClient()
	if err != nil {
		return nil, nil, nil, nil, "", errors.Wrap(err, "unable to create Tekton client")
	}

	jxClient, _, err := factory.CreateJXClient()
	if err != nil {
		return nil, nil, nil, nil, "", errors.Wrap(err, "unable to create JX client")
	}

	kubeClient, ns, err := factory.CreateKubeClient()
	if err != nil {
		return nil, nil, nil, nil, "", errors.Wrap(err, "unable to create Kube client")
	}
	ns, _, err = kube.GetDevNamespace(kubeClient, ns)
	if err != nil {
		return nil, nil, nil, nil, "", errors.Wrap(err, "unable to find the dev namespace")
	}

	config, err := factory.CreateKubeConfig()
	if err != nil {
		return nil, nil, nil, nil, "", errors.Wrap(err, "unable to create kubeconfig for Lighthouse client")
	}
	lhClient, err := clientset.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, nil, "", errors.Wrap(err, "unable to create Lighthouse client")
	}

	return tektonClient, jxClient, kubeClient, lhClient, ns, nil
}
