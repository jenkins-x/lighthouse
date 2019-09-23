package clients

import (
	"github.com/pkg/errors"

	jxclient "github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/jxfactory"
	"github.com/jenkins-x/jx/pkg/kube"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	kubeclient "k8s.io/client-go/kubernetes"
)

// GetClientsAndNamespace returns the tekton, jx and kube clients and the dev namespace
func GetClientsAndNamespace() (tektonclient.Interface, jxclient.Interface, kubeclient.Interface, string, error) {
	factory := jxfactory.NewFactory()

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
