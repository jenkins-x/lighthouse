package merge

import (
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/config"
)

// CombineConfigs combines the two configurations together from multiple files
func CombineConfigs(a, b *v1alpha1.RepositoryConfig) *v1alpha1.RepositoryConfig {
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
	if b.Spec.BranchProtection != nil {
		if a.Spec.BranchProtection == nil {
			a.Spec.BranchProtection = &config.ContextPolicy{}
		}
		for _, s := range b.Spec.BranchProtection.Contexts {
			a.Spec.BranchProtection.Contexts = append(a.Spec.BranchProtection.Contexts, s)
		}
	}
	return a
}
