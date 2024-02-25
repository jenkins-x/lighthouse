package inrepo

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/lighthouse/pkg/util"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/jenkins-x/lighthouse/pkg/filebrowser"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/triggerconfig"
	"github.com/jenkins-x/lighthouse/pkg/triggerconfig/merge"
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

// MergeTriggers merges the configuration with any `lighthouse.yaml` files in the repository
func MergeTriggers(cfg *config.Config, pluginCfg *plugins.Configuration, fileBrowsers *filebrowser.FileBrowsers, fc filebrowser.FetchCache, cache *ResolverCache, ownerName string, repoName string, sha string) (bool, error) {
	repoConfig, err := LoadTriggerConfig(fileBrowsers, fc, cache, ownerName, repoName, sha)
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
func LoadTriggerConfig(fileBrowsers *filebrowser.FileBrowsers, fc filebrowser.FetchCache, cache *ResolverCache, ownerName string, repoName string, sha string) (*triggerconfig.Config, error) {
	var answer *triggerconfig.Config
	err := fileBrowsers.LighthouseGitFileBrowser().WithDir(ownerName, repoName, sha, fc, []string{"/.lighthouse/**"}, func(dir string) error {
		path := filepath.Join(dir, ".lighthouse")
		exists, err := util.DirExists(path)
		if err != nil {
			return errors.Wrapf(err, "failed to check if dir exists %s", path)
		}
		m := map[string]*triggerconfig.Config{}
		if exists {
			fs, err := os.ReadDir(path)
			if err != nil {
				return errors.Wrapf(err, "failed to read dir %s", path)
			}
			for _, f := range fs {
				name := f.Name()
				if strings.HasPrefix(name, ".") {
					continue
				}
				if f.IsDir() {
					filePath := filepath.Join(path, name, "triggers.yaml")
					cfg, err := loadConfigFile(filePath, fileBrowsers, fc, cache, ownerName, repoName, filePath, sha)
					if err != nil {
						return errors.Wrapf(err, "failed to load file %s in %s/%s with sha %s", filePath, ownerName, repoName, sha)
					}
					if cfg != nil {
						m[filePath] = cfg
					}

				} else if name == "triggers.yaml" {
					filePath := filepath.Join(path, "triggers.yaml")
					cfg, err := loadConfigFile(filePath, fileBrowsers, fc, cache, ownerName, repoName, filePath, sha)
					if err != nil {
						return errors.Wrapf(err, "failed to load file %s in %s/%s with sha %s", filePath, ownerName, repoName, sha)
					}
					if cfg != nil {
						m[filePath] = cfg
					}

				}
			}
		}
		answer, err = mergeConfigs(m)
		return err
	})
	return answer, err
}

func mergeConfigs(m map[string]*triggerconfig.Config) (*triggerconfig.Config, error) {
	var answer *triggerconfig.Config

	// lets check for duplicates
	presubmitNames := map[string]string{}
	postsubmitNames := map[string]string{}
	periodicNames := map[string]string{}
	deploymentNames := map[string]string{}
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
		for _, ps := range cfg.Spec.Periodics {
			name := ps.Name
			otherFile := periodicNames[name]
			if otherFile == "" {
				periodicNames[name] = file
			} else {
				return nil, errors.Errorf("duplicate periodic %s in file %s and %s", name, otherFile, file)
			}
		}
		for _, ps := range cfg.Spec.Deployments {
			name := ps.Name
			otherFile := deploymentNames[name]
			if otherFile == "" {
				deploymentNames[name] = file
			} else {
				return nil, errors.Errorf("duplicate deployment %s in file %s and %s", name, otherFile, file)
			}
		}
		answer = merge.CombineConfigs(answer, cfg)
	}
	if answer == nil {
		answer = &triggerconfig.Config{}
	}
	return answer, nil
}

func loadConfigFile(filePath string, fileBrowsers *filebrowser.FileBrowsers, fc filebrowser.FetchCache, cache *ResolverCache, ownerName, repoName, path, sha string) (*triggerconfig.Config, error) {
	exists, err := util.FileExists(filePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to check if file exists %s", filePath)
	}
	if !exists {
		return nil, nil
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read file %s with sha %s", filePath, sha)
	}
	if len(data) == 0 {
		return nil, nil
	}
	repoConfig := &triggerconfig.Config{}
	err = yaml.Unmarshal(data, repoConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal file %s with sha %s", filePath, sha)
	}
	dir := filepath.Dir(filePath)
	for i := range repoConfig.Spec.Presubmits {
		r := &repoConfig.Spec.Presubmits[i]
		sourcePath := r.SourcePath
		if sourcePath != "" {
			if r.Agent == "" {
				r.Agent = job.TektonPipelineAgent
			}
			// lets load the local file data now as we have locked the git file system
			data, err := loadLocalFile(dir, sourcePath, sha)
			if err != nil {
				return nil, err
			}
			r.SetPipelineLoader(func(base *job.Base) error {
				err = loadJobBaseFromSourcePath(data, fileBrowsers, fc, cache, base, ownerName, repoName, sourcePath, sha)
				if err != nil {
					return errors.Wrapf(err, "failed to load source for presubmit %s", r.Name)
				}
				r.Base = *base
				if r.Agent == "" && r.PipelineRunSpec != nil {
					r.Agent = job.TektonPipelineAgent
				}
				return nil
			})
		}
	}
	for i := range repoConfig.Spec.Postsubmits {
		r := &repoConfig.Spec.Postsubmits[i]
		sourcePath := r.SourcePath
		if sourcePath != "" {
			if r.Agent == "" {
				r.Agent = job.TektonPipelineAgent
			}
			// lets load the local file data now as we have locked the git file system
			data, err := loadLocalFile(dir, sourcePath, sha)
			if err != nil {
				return nil, err
			}
			r.SetPipelineLoader(func(base *job.Base) error {
				err = loadJobBaseFromSourcePath(data, fileBrowsers, fc, cache, base, ownerName, repoName, sourcePath, sha)
				if err != nil {
					return errors.Wrapf(err, "failed to load source for postsubmit %s", r.Name)
				}
				r.Base = *base
				if r.Agent == "" && r.PipelineRunSpec != nil {
					r.Agent = job.TektonPipelineAgent
				}
				return nil
			})
		}
	}
	for i := range repoConfig.Spec.Deployments {
		r := &repoConfig.Spec.Deployments[i]
		sourcePath := r.SourcePath
		if sourcePath != "" {
			if r.Agent == "" {
				r.Agent = job.TektonPipelineAgent
			}
			// lets load the local file data now as we have locked the git file system
			data, err := loadLocalFile(dir, sourcePath, sha)
			if err != nil {
				return nil, err
			}
			r.SetPipelineLoader(func(base *job.Base) error {
				err = loadJobBaseFromSourcePath(data, fileBrowsers, fc, cache, base, ownerName, repoName, sourcePath, sha)
				if err != nil {
					return errors.Wrapf(err, "failed to load source for deployment %s", r.Name)
				}
				r.Base = *base
				if r.Agent == "" && r.PipelineRunSpec != nil {
					r.Agent = job.TektonPipelineAgent
				}
				return nil
			})
		}
	}
	for i := range repoConfig.Spec.Periodics {
		r := &repoConfig.Spec.Periodics[i]
		sourcePath := r.SourcePath
		if sourcePath != "" {
			if r.Agent == "" {
				r.Agent = job.TektonPipelineAgent
			}
			// lets load the local file data now as we have locked the git file system
			data, err := loadLocalFile(dir, sourcePath, sha)
			if err != nil {
				return nil, err
			}
			r.SetPipelineLoader(func(base *job.Base) error {
				err = loadJobBaseFromSourcePath(data, fileBrowsers, fc, cache, base, ownerName, repoName, sourcePath, sha)
				if err != nil {
					return errors.Wrapf(err, "failed to load source for periodic %s", r.Name)
				}
				r.Base = *base
				if r.Agent == "" && r.PipelineRunSpec != nil {
					r.Agent = job.TektonPipelineAgent
				}
				return nil
			})
		}
	}

	return repoConfig, nil
}

func loadLocalFile(dir, name, sha string) ([]byte, error) {
	path := filepath.Join(dir, name)
	exists, err := util.FileExists(path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find file %s", path)
	}
	if exists {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read file %s with sha %s", path, sha)
		}
		return data, nil
	}
	return nil, nil
}

func loadJobBaseFromSourcePath(data []byte, fileBrowsers *filebrowser.FileBrowsers, fc filebrowser.FetchCache, cache *ResolverCache, j *job.Base, ownerName, repoName, path, sha string) error {
	if data == nil {
		_, err := url.ParseRequestURI(path)
		if err == nil {
			data, err = getPipelineFromURL(path)
			if err != nil {
				return errors.Wrapf(err, "failed to get pipeline from URL %s ", path)
			}
		} else {
			return errors.Errorf("file does not exist and not a URL: %s", path)
		}
	}
	if len(data) == 0 {
		return errors.Errorf("empty file file %s in repo %s/%s for sha %s", path, ownerName, repoName, sha)
	}

	if strings.Contains(string(data), "image: uses:") {
		j.IsResolvedWithUsesSyntax = true
	}

	dir := filepath.Dir(path)

	message := fmt.Sprintf("in repo %s/%s with sha %s", ownerName, repoName, sha)

	usesResolver := &UsesResolver{
		FileBrowsers: fileBrowsers,
		FetchCache:   fc,
		Cache:        cache,
		OwnerName:    ownerName,
		RepoName:     repoName,
		SHA:          sha,
		Dir:          dir,
		Message:      message,
	}

	prs, err := LoadTektonResourceAsPipelineRun(usesResolver, data)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal YAML file %s in repo %s/%s with sha %s", path, ownerName, repoName, sha)
	}
	j.PipelineRunSpec = &prs.Spec
	return nil
}

func getPipelineFromURL(path string) ([]byte, error) {
	client := &http.Client{
		CheckRedirect: redirectPolicyFunc,
	}

	req, err := http.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get URL %s", path)
	}
	req.Header.Add("Authorization", "Basic "+basicAuthGit())

	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to request URL %s", path)
	}

	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read body from URL %s", path)
	}
	return data, nil
}

// IsScmNotFound returns true if the error is a not found error
func IsScmNotFound(err error) bool {
	if err != nil {
		// I think that we should instead rely on the http status (404)
		// until jenkins-x go-scm is updated t return that in the error this works for github and gitlab
		return strings.Contains(err.Error(), scm.ErrNotFound.Error())
	}
	return false
}

func basicAuthGit() string {
	user := os.Getenv("GIT_USER")
	token := os.Getenv("GIT_TOKEN")
	if user != "" && token != "" {
		auth := user + ":" + token
		return base64.StdEncoding.EncodeToString([]byte(auth))
	}
	return ""
}

func redirectPolicyFunc(req *http.Request, via []*http.Request) error {
	req.Header.Add("Authorization", "Basic "+basicAuthGit())
	return nil
}
