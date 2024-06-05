package clients

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

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
	kubeCfg, err := GetConfig("", "")
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

// GetConfig returns a rest.Config to be used for kubernetes client creation.
// It does so in the following order:
//  1. Use the passed kubeconfig/masterURL.
//  2. Fallback to the KUBECONFIG environment variable.
//  3. Fallback to in-cluster config.
//  4. Fallback to the ~/.kube/config.
func GetConfig(masterURL, kubeconfig string) (*rest.Config, error) {
	if kubeconfig == "" {
		kubeconfig = os.Getenv("KUBECONFIG")
	}
	// If we have an explicit indication of where the kubernetes config lives, read that.
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	}
	// If not, try the in-cluster config.
	if c, err := rest.InClusterConfig(); err == nil {
		return c, nil
	}
	// If no in-cluster config, try the default location in the user's home directory.
	if usr, err := user.Current(); err == nil {
		if c, err := clientcmd.BuildConfigFromFlags("", filepath.Join(usr.HomeDir, ".kube", "config")); err == nil {
			return c, nil
		}
	}

	return nil, fmt.Errorf("could not create a valid kubeconfig")
}
