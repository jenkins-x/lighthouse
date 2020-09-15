package scmload

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/go-scm/scm"
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
func MergeTriggers(cfg *config.Config, pluginCfg *plugins.Configuration, scmClient *scm.Client, repoOwner string, repoName string, sha string) (bool, error) {
	repoConfig, err := LoadTriggerConfig(scmClient, repoOwner, repoName, sha)
	if err != nil {
		return false, errors.Wrap(err, "failed to load configs")
	}
	if repoConfig == nil {
		return false, nil
	}

	err = merge.ConfigMerge(cfg, pluginCfg, repoConfig, repoOwner, repoName)
	if err != nil {
		return false, errors.Wrapf(err, "failed to merge repository config with repository %s/%s and sha %s", repoOwner, repoName, sha)
	}
	return true, nil
}

// LoadTriggerConfig loads the `lighthouse.yaml` configuration files in the repository
func LoadTriggerConfig(scmClient *scm.Client, repoOwner string, repoName string, sha string) (*triggerconfig.Config, error) {
	if sha == "" {
		sha = "master"
	}

	repo := scm.Join(repoOwner, repoName)

	m := map[string]*triggerconfig.Config{}

	ctx := context.Background()
	path := ".lighthouse"
	files, _, err := scmClient.Contents.List(ctx, repo, path, sha)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find any lighthouse configuration files in repo %s at sha %s", repo, sha)
	}
	for _, f := range files {
		if isDirType(f.Type) {
			filePath := path + "/" + f.Name + "/triggers.yaml"
			cfg, err := loadConfigFile(ctx, scmClient, repo, filePath, sha)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to load file %s in %s with sha %s", filePath, repo, sha)
			}
			if cfg != nil {
				m[filePath] = cfg
			}
		} else if f.Name == "trigger.yaml" {
			filePath := path + "/" + f.Name
			cfg, err := loadConfigFile(ctx, scmClient, repo, filePath, sha)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to load file %s in %s with sha %s", filePath, repo, sha)
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

func loadConfigFile(ctx context.Context, client *scm.Client, repo string, path string, sha string) (*triggerconfig.Config, error) {
	c, r, err := client.Contents.Find(ctx, repo, path, sha)
	if err != nil {
		if r != nil && r.Status == 404 {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to find file %s in repo %s with sha %s status %d", path, repo, sha, r.Status)
	}
	if len(c.Data) == 0 {
		return nil, nil
	}
	repoConfig := &triggerconfig.Config{}
	err = yaml.Unmarshal(c.Data, repoConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal file %s in repo %s with sha %s", path, repo, sha)
	}
	dir := filepath.Dir(path)
	for i := range repoConfig.Spec.Presubmits {
		r := &repoConfig.Spec.Presubmits[i]
		if r.SourcePath != "" {
			err = loadJobBaseFromSourcePath(ctx, client, &r.Base, repo, filepath.Join(dir, r.SourcePath), sha)
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
			err = loadJobBaseFromSourcePath(ctx, client, &r.Base, repo, filepath.Join(dir, r.SourcePath), sha)
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

func loadJobBaseFromSourcePath(ctx context.Context, client *scm.Client, j *job.Base, repo, path, sha string) error {
	c, r, err := client.Contents.Find(ctx, repo, path, sha)
	if err != nil {
		if r != nil && r.Status == 404 {
			return errors.Errorf("no file %s in repo %s for sha %s", path, repo, sha)
		}
		return errors.Wrapf(err, "failed to find file %s in repo %s with sha %s status %d", path, repo, sha, r.Status)
	}
	if len(c.Data) == 0 {
		return errors.Errorf("empty file file %s in repo %s for sha %s", path, repo, sha)
	}

	// for now lets assume its a PipelineRun we could eventually support different kinds
	prs := &tektonv1beta1.PipelineRun{}
	err = yaml.Unmarshal(c.Data, prs)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal YAML file %s in repo %s with sha %s", path, repo, sha)
	}
	j.PipelineRunSpec = &prs.Spec
	return nil
}
