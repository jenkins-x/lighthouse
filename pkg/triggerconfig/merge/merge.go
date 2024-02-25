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
		// lets make a new map to avoid concurrent modifications
		m := map[string][]job.Presubmit{}
		if cfg.Presubmits != nil {
			for k, v := range cfg.Presubmits {
				m[k] = append([]job.Presubmit{}, v...)
			}
		}
		cfg.Presubmits = m

		ps := cfg.Presubmits[repoKey]
		for _, p := range repoConfig.Spec.Presubmits {
			found := false
			for i := range ps {
				pt2 := &ps[i]
				if pt2.Name == p.Name {
					*pt2 = p
					found = true
					break
				}
			}
			if !found {
				ps = append(ps, p)
			}
		}
		cfg.Presubmits[repoKey] = ps
	}
	if len(repoConfig.Spec.Postsubmits) > 0 {
		// lets make a new map to avoid concurrent modifications
		m := map[string][]job.Postsubmit{}
		if cfg.Postsubmits != nil {
			for k, v := range cfg.Postsubmits {
				m[k] = append([]job.Postsubmit{}, v...)
			}
		}
		cfg.Postsubmits = m

		ps := cfg.Postsubmits[repoKey]
		for _, p := range repoConfig.Spec.Postsubmits {
			found := false
			for i := range ps {
				pt2 := &ps[i]
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
	if len(repoConfig.Spec.Periodics) > 0 {
		cfg.Periodics = append(cfg.Periodics, repoConfig.Spec.Periodics...)
	}
	if len(repoConfig.Spec.Deployments) > 0 {
		// lets make a new map to avoid concurrent modifications
		m := map[string][]job.Deployment{}
		if cfg.Deployments != nil {
			for k, v := range cfg.Deployments {
				m[k] = append([]job.Deployment{}, v...)
			}
		}
		cfg.Deployments = m

		ps := cfg.Deployments[repoKey]
		for _, p := range repoConfig.Spec.Deployments {
			found := false
			for i := range ps {
				pt2 := &ps[i]
				if pt2.Name == p.Name {
					ps[i] = p
					found = true
				}
			}
			if !found {
				ps = append(ps, p)
			}
		}
		cfg.Deployments[repoKey] = ps
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
	migrateOldConfig(&cfg.JobConfig)
	err = cfg.Init(cfg.ProwConfig)
	if err != nil {
		return errors.Wrapf(err, "failed to initialize config")
	}
	err = cfg.Validate(cfg.ProwConfig)
	if err != nil {
		return errors.Wrapf(err, "failed to validate config")
	}
	return nil
}

// migrateOldConfig lets handle some old incorrect configuration where the trigger and rerun_command values were not setup properly
func migrateOldConfig(cfg *job.Config) {
	for _, ps := range cfg.Presubmits {
		for i := range ps {
			presubmit := &ps[i]
			if presubmit.Trigger == "/test" && presubmit.RerunCommand == "/retest" {
				presubmit.Trigger = "(?m)^/test,?($|\\s.*)"
				presubmit.RerunCommand = "/test"
			} else if presubmit.Trigger == "/lint" && presubmit.RerunCommand == "/relint" {
				presubmit.Trigger = "(?m)^/lint,?($|\\s.*)"
				presubmit.RerunCommand = "/lint"
			}
		}
	}
}
