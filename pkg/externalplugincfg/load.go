package externalplugincfg

import (
	"context"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
	"strings"
)

func LoadDisabledPlugins(l *logrus.Entry, client kubernetes.Interface, namespace string) ([]string, error) {
	answer := []string{}

	ctx := context.TODO()

	l = l.WithFields(map[string]interface{}{
		"Namespace": namespace,
		"ConfigMap": ConfigMapName,
	})
	cm, err := client.CoreV1().ConfigMaps(namespace).Get(ctx, ConfigMapName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			l.Warnf("no ConfigMap for the external plugin configuration")
			return answer, nil
		}
		if err != nil {
			return nil, errors.Wrapf(err, "failed to load ConfigMap %s in namespace %s", ConfigMapName, namespace)
		}
	}
	text := ""
	if cm != nil && cm.Data != nil {
		text = cm.Data["config.yaml"]
	}
	if text == "" {
		l.Warnf("no config.yaml entry in external plugin ConfigMap")
		return answer, nil
	}

	var externalPlugins []ExternalPlugin
	err = yaml.Unmarshal([]byte(text), &externalPlugins)
	if err != nil {
		return answer, errors.Wrapf(err, "failed to unmarshal external plugin configuration")
	}

	for _, p := range externalPlugins {
		if util.StringArrayIndex(answer, p.Name) >= 0 {
			continue
		}
		for _, r := range p.RequiredResources {
			exists, err := CheckResourceExists(ctx, client, namespace, r)
			if err != nil {
				return answer, errors.Wrapf(err, "failed to check if resource exists %s", r.String())
			}
			if !exists {
				l.WithField("ExternalPlugin", p.Name).Infof("disabling external plugin as resource is not available %s", r.String())
				answer = append(answer, p.Name)
				break
			}
		}
	}
	return answer, nil
}

// CheckResourceExists checks if a service exists
func CheckResourceExists(ctx context.Context, client kubernetes.Interface, namespace string, r Resource) (bool, error) {
	k := strings.ToLower(r.Kind)
	ns := r.Namespace
	if ns == "" {
		ns = namespace
	}
	name := r.Name
	if name == "" {
		return true, nil
	}
	if k == "service" {
		_, err := client.CoreV1().Services(ns).Get(ctx, name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		if err != nil {
			return false, errors.Wrapf(err, "failed to find Service %s/%s", ns, name)
		}
		return true, nil
	}
	// TODO
	return true, nil
}
