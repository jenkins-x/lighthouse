package merge

import (
	"github.com/jenkins-x/lighthouse/pkg/config/branchprotection"
	"github.com/jenkins-x/lighthouse/pkg/repoconfig"
)

// CombineConfigs combines the two configurations together from multiple files
func CombineConfigs(a, b *repoconfig.RepositoryConfig) *repoconfig.RepositoryConfig {
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
			a.Spec.BranchProtection = &branchprotection.ContextPolicy{}
		}
		for _, s := range b.Spec.BranchProtection.Contexts {
			a.Spec.BranchProtection.Contexts = append(a.Spec.BranchProtection.Contexts, s)
		}
	}
	return a
}
