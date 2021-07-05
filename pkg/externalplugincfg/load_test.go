package externalplugincfg_test

import (
	"testing"

	"github.com/jenkins-x/lighthouse/pkg/externalplugincfg"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

var testYaml = `- name: cd-indicators
  requiredResources:
  - kind: Service
    name: cd-indicators
    namespace: jx
- name: lighthouse-webui-plugin
  requiredResources:
  - kind: Service
    name: lighthouse-webui-plugin
    namespace: jx`

func TestLoadExternalPluginConfig(t *testing.T) {
	ns := "jx"

	expected := []string{"lighthouse-webui-plugin"}
	l := logrus.StandardLogger().WithField("Testing", true)

	client := fake.NewSimpleClientset(
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      externalplugincfg.ConfigMapName,
			},
			Data: map[string]string{
				"config.yaml": testYaml,
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      "cd-indicators",
			},
		},
	)

	got, err := externalplugincfg.LoadDisabledPlugins(l, client, ns)
	require.NoError(t, err, "failed to load disabled external plugins")
	assert.Equal(t, expected, got, "for disabled external plugins")
}
