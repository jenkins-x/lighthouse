package merge

import (
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
)

// ConfigMerge merges the repository configuration into the global configuration
func ConfigMerge(cfg *config.Config, pluginsCfg *plugins.Configuration, repoConfig *v1alpha1.RepositoryConfig, repoOwner string, repoName string) error {
	repoKey := repoOwner + "/" + repoName
	if len(repoConfig.Spec.Presubmits) > 0 {
		if cfg.Presubmits == nil {
			cfg.Presubmits = map[string][]config.Presubmit{}
		}

		var ps []config.Presubmit
		for _, p := range repoConfig.Spec.Presubmits {
			ps = append(ps, ToPresubmit(p))
		}
		cfg.Presubmits[repoKey] = ps
	}
	if len(repoConfig.Spec.Postsubmits) > 0 {
		if cfg.Postsubmits == nil {
			cfg.Postsubmits = map[string][]config.Postsubmit{}
		}

		var ps []config.Postsubmit
		for _, p := range repoConfig.Spec.Postsubmits {
			ps = append(ps, ToPostsubmit(p))
		}
		cfg.Postsubmits[repoKey] = ps
	}

	if repoConfig.Spec.BranchProtection != nil {
		if cfg.BranchProtection.Orgs == nil {
			cfg.BranchProtection.Orgs = map[string]config.Org{}
		}
		org := cfg.BranchProtection.Orgs[repoOwner]
		if org.Repos == nil {
			org.Repos = map[string]config.Repo{}
		}
		repo := org.Repos[repoName]
		repo.RequiredStatusChecks = repoConfig.Spec.BranchProtection
		org.Repos[repoName] = repo
		cfg.BranchProtection.Orgs[repoOwner] = org
	}

	keeper := repoConfig.Spec.Keeper
	if keeper != nil {
		if keeper.Query != nil {
			for i, q := range cfg.Keeper.Queries {
				// remove the old repository
				idx := StringArrayIndex(q.Repos, repoKey)
				if idx >= 0 {
					cfg.Keeper.Queries[i].Repos = RemoveStringArrayAtIndex(cfg.Keeper.Queries[i].Repos, idx)
					break
				}
			}
			query := *keeper.Query
			query.Repos = []string{repoKey}
			cfg.Keeper.Queries = append(cfg.Keeper.Queries, query)
		}
	}

	pc := repoConfig.Spec.PluginConfig
	if pc != nil {
		if pluginsCfg.Plugins == nil {
			pluginsCfg.Plugins = map[string][]string{}
		}
		pluginsCfg.Plugins[repoKey] = pc.Plugins

		if pc.Approve != nil {
			pluginsCfg.Approve = append(pluginsCfg.Approve, ToApprove(pc.Approve, repoKey))
		}

		// lets add to the last idx
		idx := len(pluginsCfg.Triggers) - 1
		if idx < 0 {
			idx = 0
			pluginsCfg.Triggers = append(pluginsCfg.Triggers, plugins.Trigger{})
		}
		if StringArrayIndex(pluginsCfg.Triggers[idx].Repos, repoKey) < 0 {
			pluginsCfg.Triggers[idx].Repos = append(pluginsCfg.Triggers[idx].Repos, repoKey)
		}
	}
	return nil
}
