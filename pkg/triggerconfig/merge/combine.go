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
	for _, r := range b.Spec.Presubmits {
		a.Spec.Presubmits = append(a.Spec.Presubmits, r)
	}
	for _, r := range b.Spec.Postsubmits {
		a.Spec.Postsubmits = append(a.Spec.Postsubmits, r)
	}
	return a
}
