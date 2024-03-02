package merge

import (
	"github.com/jenkins-x/lighthouse/pkg/triggerconfig"
)

// CombineConfigs combines the two configurations together from multiple files
func CombineConfigs(a, b *triggerconfig.Config) *triggerconfig.Config {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	a.Spec.Presubmits = append(a.Spec.Presubmits, b.Spec.Presubmits...)
	a.Spec.Postsubmits = append(a.Spec.Postsubmits, b.Spec.Postsubmits...)
	a.Spec.Periodics = append(a.Spec.Periodics, b.Spec.Periodics...)
	a.Spec.Deployments = append(a.Spec.Deployments, b.Spec.Deployments...)
	return a
}
