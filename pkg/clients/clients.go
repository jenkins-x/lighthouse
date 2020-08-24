package clients

import (
	clientset "github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	kubeclient "k8s.io/client-go/kubernetes"

	//  import the auth plugin package - see https://github.com/jenkins-x/lighthouse/issues/928
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

// GetAPIClients returns the tekton, kube, and Lighthouse clients and the kubeconfig used to create them
func GetAPIClients() (tektonclient.Interface, kubeclient.Interface, clientset.Interface, *rest.Config, error) {
	kubeCfg, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, "unable to get kubeconfig")
	}

	lhClient, err := clientset.NewForConfig(kubeCfg)
	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, "unable to create Lighthouse client")
	}

	tektonClient, err := tektonclient.NewForConfig(kubeCfg)
	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, "unable to create Tekton client")
	}

	kubeClient, err := kubeclient.NewForConfig(kubeCfg)
	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, "unable to create Kubernetes client")
	}

	return tektonClient, kubeClient, lhClient, kubeCfg, nil
}
