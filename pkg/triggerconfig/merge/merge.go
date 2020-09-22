package merge

import (
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/triggerconfig"
	"github.com/pkg/errors"
)

// ConfigMerge merges the repository configuration into the global configuration
func ConfigMerge(cfg *config.Config, pluginsCfg *plugins.Configuration, repoConfig *triggerconfig.Config, repoOwner string, repoName string) error {
	repoKey := repoOwner + "/" + repoName
	if len(repoConfig.Spec.Presubmits) > 0 {
		if cfg.Presubmits == nil {
			cfg.Presubmits = map[string][]job.Presubmit{}
		}

		ps := cfg.Presubmits[repoKey]
		for i, p := range repoConfig.Spec.Presubmits {
			found := false
			for _, pt2 := range ps {
				if pt2.Name == p.Name {
					ps[i] = p
					found = true
				}
			}
			if !found {
				ps = append(ps, p)
			}
		}
		cfg.Presubmits[repoKey] = ps
	}
	if len(repoConfig.Spec.Postsubmits) > 0 {
		if cfg.Postsubmits == nil {
			cfg.Postsubmits = map[string][]job.Postsubmit{}
		}

		ps := cfg.Postsubmits[repoKey]
		for i, p := range repoConfig.Spec.Postsubmits {
			found := false
			for _, pt2 := range ps {
				if pt2.Name == p.Name {
					ps[i] = p
					found = true
				}
			}
			if !found {
				ps = append(ps, p)
			}
		}
		cfg.Postsubmits[repoKey] = ps
	}

	// lets make sure we've got a trigger added
	idx := len(pluginsCfg.Triggers) - 1
	if idx < 0 {
		idx = 0
		pluginsCfg.Triggers = append(pluginsCfg.Triggers, plugins.Trigger{})
	}
	if StringArrayIndex(pluginsCfg.Triggers[idx].Repos, repoKey) < 0 {
		pluginsCfg.Triggers[idx].Repos = append(pluginsCfg.Triggers[idx].Repos, repoKey)
	}

	// lets validate the configuration is valid
	err := pluginsCfg.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate plugins")
	}
	err = cfg.Validate(cfg.ProwConfig)
	if err != nil {
		return errors.Wrapf(err, "failed to validate config")
	}
	return nil
}
