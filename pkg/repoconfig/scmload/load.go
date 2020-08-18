package scmload

import (
	"context"
	"strings"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/repoconfig"
	"github.com/jenkins-x/lighthouse/pkg/repoconfig/merge"
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

// MergeConfig merges the configuration with any `lighthouse.yaml` files in the repository
func MergeConfig(cfg *config.Config, pluginCfg *plugins.Configuration, scmClient *scm.Client, repoOwner string, repoName string, sha string) (bool, error) {
	repoConfig, err := LoadRepositoryConfig(scmClient, repoOwner, repoName, sha)
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

// LoadRepositoryConfig loads the `lighthouse.yaml` configuration files in the repository
func LoadRepositoryConfig(scmClient *scm.Client, repoOwner string, repoName string, sha string) (*repoconfig.RepositoryConfig, error) {
	if sha == "" {
		sha = "master"
	}

	var answer *repoconfig.RepositoryConfig
	repo := scm.Join(repoOwner, repoName)

	ctx := context.Background()
	path := ".lighthouse"
	files, _, err := scmClient.Contents.List(ctx, repo, path, sha)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find any lighthouse configuration files in repo %s at sha %s", repo, sha)
	}
	for _, f := range files {
		if isDirType(f.Type) {
			filePath := path + "/" + f.Name + "/lighthouse.yaml"
			cfg, err := loadConfigFile(ctx, scmClient, repo, filePath, sha)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to load file %s in %s with sha %s", filePath, repo, sha)
			}
			if cfg != nil {
				answer = merge.CombineConfigs(answer, cfg)
			}
		} else if f.Name == "lighthouse.yaml" {
			filePath := path + "/" + f.Name
			cfg, err := loadConfigFile(ctx, scmClient, repo, filePath, sha)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to load file %s in %s with sha %s", filePath, repo, sha)
			}
			if cfg != nil {
				answer = merge.CombineConfigs(answer, cfg)
			}
		}
	}
	return answer, nil
}

func isDirType(t string) bool {
	return strings.ToLower(t) == "dir"
}

func loadConfigFile(ctx context.Context, client *scm.Client, repo string, path string, sha string) (*repoconfig.RepositoryConfig, error) {
	c, r, err := client.Contents.Find(ctx, repo, path, sha)
	if err != nil {
		if r != nil && r.Status == 404 {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to find  file %s in repo %s with sha %s status %d", path, repo, sha, r.Status)
	}
	if len(c.Data) == 0 {
		return nil, nil
	}
	repoConfig := &repoconfig.RepositoryConfig{}
	err = yaml.Unmarshal(c.Data, repoConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal file %s in repo %s with sha %s", path, repo, sha)
	}
	return repoConfig, nil
}
