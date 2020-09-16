package inrepo

import (
	"path/filepath"
	"strings"

	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/triggerconfig"
	"github.com/jenkins-x/lighthouse/pkg/triggerconfig/merge"
	"github.com/pkg/errors"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"sigs.k8s.io/yaml"
)

// MergeTriggers merges the configuration with any `lighthouse.yaml` files in the repository
func MergeTriggers(cfg *config.Config, pluginCfg *plugins.Configuration, scmClient scmProviderClient, ownerName string, repoName string, sha string) (bool, error) {
	repoConfig, err := LoadTriggerConfig(scmClient, ownerName, repoName, sha)
	if err != nil {
		return false, errors.Wrap(err, "failed to load configs")
	}
	if repoConfig == nil {
		return false, nil
	}

	err = merge.ConfigMerge(cfg, pluginCfg, repoConfig, ownerName, repoName)
	if err != nil {
		return false, errors.Wrapf(err, "failed to merge repository config with repository %s/%s and sha %s", ownerName, repoName, sha)
	}
	return true, nil
}

// LoadTriggerConfig loads the `lighthouse.yaml` configuration files in the repository
func LoadTriggerConfig(scmClient scmProviderClient, ownerName string, repoName string, sha string) (*triggerconfig.Config, error) {
	if sha == "" {
		sha = "master"
	}

	m := map[string]*triggerconfig.Config{}

	path := ".lighthouse"
	files, err := scmClient.ListFiles(ownerName, repoName, path, sha)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find any lighthouse configuration files in repo %s/%s at sha %s", ownerName, repoName, sha)
	}
	for _, f := range files {
		if isDirType(f.Type) {
			filePath := path + "/" + f.Name + "/triggers.yaml"
			cfg, err := loadConfigFile(scmClient, ownerName, repoName, filePath, sha)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to load file %s in %s/%s with sha %s", filePath, ownerName, repoName, sha)
			}
			if cfg != nil {
				m[filePath] = cfg
			}
		} else if f.Name == "trigger.yaml" {
			filePath := path + "/" + f.Name
			cfg, err := loadConfigFile(scmClient, ownerName, repoName, filePath, sha)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to load file %s in %s/%s with sha %s", filePath, ownerName, repoName, sha)
			}
			if cfg != nil {
				m[filePath] = cfg
			}
		}
	}
	return mergeConfigs(m)
}

func mergeConfigs(m map[string]*triggerconfig.Config) (*triggerconfig.Config, error) {
	var answer *triggerconfig.Config

	// lets check for duplicates
	presubmitNames := map[string]string{}
	postsubmitNames := map[string]string{}
	for file, cfg := range m {
		for _, ps := range cfg.Spec.Presubmits {
			name := ps.Name
			otherFile := presubmitNames[name]
			if otherFile == "" {
				presubmitNames[name] = file
			} else {
				return nil, errors.Errorf("duplicate presubmit %s in file %s and %s", name, otherFile, file)
			}
		}
		for _, ps := range cfg.Spec.Postsubmits {
			name := ps.Name
			otherFile := postsubmitNames[name]
			if otherFile == "" {
				postsubmitNames[name] = file
			} else {
				return nil, errors.Errorf("duplicate postsubmit %s in file %s and %s", name, otherFile, file)
			}
		}
		answer = merge.CombineConfigs(answer, cfg)
	}
	return answer, nil
}

func isDirType(t string) bool {
	return strings.ToLower(t) == "dir"
}

func loadConfigFile(client scmProviderClient, ownerName, repoName, path, sha string) (*triggerconfig.Config, error) {
	data, err := client.GetFile(ownerName, repoName, path, sha)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find file %s in repo %s/%s with sha %s", path, ownerName, repoName, sha)
	}
	if len(data) == 0 {
		return nil, nil
	}
	repoConfig := &triggerconfig.Config{}
	err = yaml.Unmarshal(data, repoConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal file %s in repo %s/%s with sha %s", path, ownerName, repoName, sha)
	}
	dir := filepath.Dir(path)
	for i := range repoConfig.Spec.Presubmits {
		r := &repoConfig.Spec.Presubmits[i]
		if r.SourcePath != "" {
			err = loadJobBaseFromSourcePath(client, &r.Base, ownerName, repoName, filepath.Join(dir, r.SourcePath), sha)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to load Source for Presubmit %d", i)
			}

		}
		if r.Agent == "" && r.PipelineRunSpec != nil {
			r.Agent = job.TektonPipelineAgent
		}
	}
	for i := range repoConfig.Spec.Postsubmits {
		r := &repoConfig.Spec.Postsubmits[i]
		if r.SourcePath != "" {
			err = loadJobBaseFromSourcePath(client, &r.Base, ownerName, repoName, filepath.Join(dir, r.SourcePath), sha)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to load Source for Presubmit %d", i)
			}
		}
		if r.Agent == "" && r.PipelineRunSpec != nil {
			r.Agent = job.TektonPipelineAgent
		}
	}
	return repoConfig, nil
}

func loadJobBaseFromSourcePath(client scmProviderClient, j *job.Base, ownerName, repoName, path, sha string) error {
	data, err := client.GetFile(ownerName, repoName, path, sha)
	if err != nil {
		return errors.Wrapf(err, "failed to find file %s in repo %s/%s with sha %s", path, ownerName, repoName, sha)
	}
	if len(data) == 0 {
		return errors.Errorf("empty file file %s in repo %s/%s for sha %s", path, ownerName, repoName, sha)
	}

	// for now lets assume its a PipelineRun we could eventually support different kinds
	prs := &tektonv1beta1.PipelineRun{}
	err = yaml.Unmarshal(data, prs)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal YAML file %s/%s in repo %s with sha %s", path, ownerName, repoName, sha)
	}
	j.PipelineRunSpec = &prs.Spec
	return nil
}
