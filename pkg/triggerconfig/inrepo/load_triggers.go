package inrepo

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

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
	m := map[string]*triggerconfig.Config{}
	path := ".lighthouse"
	files, err := fileBrowsers.LighthouseGitFileBrowser().ListFiles(ownerName, repoName, path, sha, fc)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find any lighthouse configuration files in repo %s/%s at sha %s", ownerName, repoName, sha)
	}
	for _, f := range files {
		if isDirType(f.Type) {
			filePath := path + "/" + f.Name + "/triggers.yaml"
			cfg, err := loadConfigFile(fileBrowsers, fc, cache, ownerName, repoName, filePath, sha)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to load file %s in %s/%s with sha %s", filePath, ownerName, repoName, sha)
			}
			if cfg != nil {
				m[filePath] = cfg
			}
		} else if f.Name == "triggers.yaml" {
			filePath := path + "/" + f.Name
			cfg, err := loadConfigFile(fileBrowsers, fc, cache, ownerName, repoName, filePath, sha)
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
	if answer == nil {
		answer = &triggerconfig.Config{}
	}
	return answer, nil
}

func isDirType(t string) bool {
	return strings.ToLower(t) == "dir"
}

func loadConfigFile(fileBrowsers *filebrowser.FileBrowsers, fc filebrowser.FetchCache, cache *ResolverCache, ownerName, repoName, path, sha string) (*triggerconfig.Config, error) {
	data, err := fileBrowsers.LighthouseGitFileBrowser().GetFile(ownerName, repoName, path, sha, fc)
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
			sourcePath := r.SourcePath
			_, err := url.ParseRequestURI(sourcePath)
			if err != nil {
				sourcePath = filepath.Join(dir, r.SourcePath)
			}
			err = loadJobBaseFromSourcePath(fileBrowsers, fc, cache, &r.Base, ownerName, repoName, sourcePath, sha)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to load source for presubmit %s", r.Name)
			}

		}
		if r.Agent == "" && r.PipelineRunSpec != nil {
			r.Agent = job.TektonPipelineAgent
		}
	}
	for i := range repoConfig.Spec.Postsubmits {
		r := &repoConfig.Spec.Postsubmits[i]
		if r.SourcePath != "" {
			sourcePath := r.SourcePath
			_, err := url.ParseRequestURI(sourcePath)
			if err != nil {
				sourcePath = filepath.Join(dir, r.SourcePath)
			}
			err = loadJobBaseFromSourcePath(fileBrowsers, fc, cache, &r.Base, ownerName, repoName, sourcePath, sha)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to load source for postsubmit %s", r.Name)
			}
		}
		if r.Agent == "" && r.PipelineRunSpec != nil {
			r.Agent = job.TektonPipelineAgent
		}
	}
	return repoConfig, nil
}

func loadJobBaseFromSourcePath(fileBrowsers *filebrowser.FileBrowsers, fc filebrowser.FetchCache, cache *ResolverCache, j *job.Base, ownerName, repoName, path, sha string) error {
	var data []byte

	// source path can either be a local file or a Git URL
	_, err := url.ParseRequestURI(path)
	if err == nil {
		data, err = getPipelineFromURL(path)
		if err != nil {
			return errors.Wrapf(err, "failed to get pipeline from URL %s ", path)
		}

	} else {
		data, err = fileBrowsers.LighthouseGitFileBrowser().GetFile(ownerName, repoName, path, sha, fc)
		if err != nil {
			return errors.Wrapf(err, "failed to find file %s in repo %s/%s with sha %s", path, ownerName, repoName, sha)
		}
	}

	if len(data) == 0 {
		return errors.Errorf("empty file file %s in repo %s/%s for sha %s", path, ownerName, repoName, sha)
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

	data, err := ioutil.ReadAll(resp.Body)
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
