/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package updateconfig

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/jenkins-x/go-scm/scm"
	config2 "github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/util"
	zglob "github.com/mattn/go-zglob"
	errors2 "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	coreapi "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/yaml"
)

const (
	pluginName                     = "config-updater"
	configUpdaterContextName       = "Lighthouse Config Updater"
	configUpdaterContextMsgFailed  = "Validation errors in config map file(s)"
	configUpdaterContextMsgSuccess = "Validation successful"
	configUpdaterMsgPruneMatch     = "Validation error founds in config map file(s):"
)

func init() {
	plugins.RegisterPlugin(
		pluginName,
		plugins.Plugin{
			Description:        "The config-updater plugin automatically redeploys configuration and plugin configuration files when they change. The plugin watches for pull request merges that modify either of the config files and updates the cluster's configmap resources in response.",
			ConfigHelpProvider: configHelp,
			PullRequestHandler: handlePullRequest,
		},
	)
}

func configHelp(config *plugins.Configuration, enabledRepos []string) (map[string]string, error) {
	var configInfo map[string]string
	if len(enabledRepos) == 1 {
		msg := fmt.Sprintf(
			"The main configuration is kept in sync with '%s/%s'.\nThe plugin configuration is kept in sync with '%s/%s'.",
			enabledRepos[0],
			"prow/config.yaml",
			enabledRepos[0],
			"prow/plugins.yaml",
		)
		configInfo = map[string]string{"": msg}
	}
	return configInfo, nil
}

type scmProviderClient interface {
	CreateComment(owner, repo string, number int, isPR bool, comment string) error
	CreateStatus(org, repo, ref string, s *scm.StatusInput) (*scm.Status, error)
	GetPullRequestChanges(org, repo string, number int) ([]*scm.Change, error)
	GetFile(org, repo, filepath, commit string) ([]byte, error)
	GetPullRequest(org, repo string, number int) (*scm.PullRequest, error)
	ProviderType() string
}

type commentPruner interface {
	PruneComments(pr bool, shouldPrune func(*scm.Comment) bool)
}

func handlePullRequest(pc plugins.Agent, pre scm.PullRequestHook) error {
	cp, err := pc.CommentPruner()
	if err != nil {
		return err
	}

	return handle(pc.SCMProviderClient, pc.KubernetesClient.CoreV1(), cp, pc.Config.LighthouseJobNamespace, pc.Logger, pre, pc.PluginConfig.ConfigUpdater)
}

// FileGetter knows how to get the contents of a file by name
type FileGetter interface {
	GetFile(filename string) ([]byte, error)
}

type scmFileGetter struct {
	org, repo, commit string
	client            scmProviderClient
}

func (g *scmFileGetter) GetFile(filename string) ([]byte, error) {
	return g.client.GetFile(g.org, g.repo, filename, g.commit)
}

type configValidateResults struct {
	cmName string
	err    error
}

// Update updates the configmap with the data from the identified files
func Update(fg FileGetter, kc corev1.ConfigMapInterface, name, namespace string, updates []ConfigMapUpdate, logger *logrus.Entry) error {
	cm, getErr := kc.Get(context.TODO(), name, metav1.GetOptions{})
	isNotFound := errors.IsNotFound(getErr)
	if getErr != nil && !isNotFound {
		return fmt.Errorf("failed to fetch current state of configmap: %v", getErr)
	}

	if cm == nil {
		cm = &coreapi.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}
	}
	if cm.Data == nil {
		cm.Data = map[string]string{}
	}
	if cm.BinaryData == nil {
		cm.BinaryData = map[string][]byte{}
	}

	for _, upd := range updates {
		if upd.Filename == "" {
			logger.WithField("key", upd.Key).Debug("Deleting key.")
			delete(cm.Data, upd.Key)
			delete(cm.BinaryData, upd.Key)
			continue
		}

		content, err := fg.GetFile(upd.Filename)
		if err != nil {
			return fmt.Errorf("get file err: %v", err)
		}
		logger.WithFields(logrus.Fields{"key": upd.Key, "cmName": upd.Filename}).Debug("Populating key.")
		value := content
		if upd.GZIP {
			buff := bytes.NewBuffer([]byte{})
			// TODO: this error is wildly unlikely for anything that
			// would actually fit in a configmap, we could just as well return
			// the error instead of falling back to the raw content
			z := gzip.NewWriter(buff)
			if _, err := z.Write(content); err != nil {
				logger.WithError(err).Error("failed to gzip content, falling back to raw")
			} else {
				if err := z.Close(); err != nil {
					logger.WithError(err).Error("failed to flush gzipped content (!?), falling back to raw")
				} else {
					value = buff.Bytes()
				}
			}
		}
		if utf8.ValidString(string(value)) {
			delete(cm.BinaryData, upd.Key)
			cm.Data[upd.Key] = string(value)
		} else {
			delete(cm.Data, upd.Key)
			cm.BinaryData[upd.Key] = value
		}
	}

	var updateErr error
	var verb string
	if getErr != nil && isNotFound {
		verb = "create"
		_, updateErr = kc.Create(context.TODO(), cm, metav1.CreateOptions{})
	} else {
		verb = "update"
		_, updateErr = kc.Update(context.TODO(), cm, metav1.UpdateOptions{})
	}
	if updateErr != nil {
		return fmt.Errorf("%s config map err: %v", verb, updateErr)
	}
	return nil
}

// ConfigMapID is a name/namespace combination that identifies a config map
type ConfigMapID struct {
	Name, Namespace string
}

// ConfigMapUpdate is populated with information about a config map that should
// be updated.
type ConfigMapUpdate struct {
	Key, Filename string
	GZIP          bool
}

// FilterChanges determines which of the changes are relevant for config updating, returning mapping of
// config map to key to cmName to update that key from.
func FilterChanges(cfg plugins.ConfigUpdater, changes []*scm.Change, log *logrus.Entry) map[ConfigMapID][]ConfigMapUpdate {
	toUpdate := map[ConfigMapID][]ConfigMapUpdate{}
	for _, change := range changes {
		var cm plugins.ConfigMapSpec
		found := false

		for key, configMap := range cfg.Maps {
			var matchErr error
			found, matchErr = zglob.Match(key, change.Path)
			if matchErr != nil {
				// Should not happen, log matchErr and continue
				log.WithError(matchErr).Info("key matching error")
				continue
			}

			if found {
				cm = configMap
				break
			}
		}

		if !found {
			continue // This file does not define a configmap
		}

		// Yes, update the configmap with the contents of this file
		for _, ns := range append(cm.Namespaces) {
			id := ConfigMapID{Name: cm.Name, Namespace: ns}
			key := cm.Key
			if key == "" {
				key = path.Base(change.Path)
				// if the key changed, we need to remove the old key
				if change.Renamed {
					oldKey := path.Base(change.PreviousPath)
					// not setting the cmName field will cause the key to be
					// deleted
					toUpdate[id] = append(toUpdate[id], ConfigMapUpdate{Key: oldKey})
				}
			}
			if change.Deleted {
				toUpdate[id] = append(toUpdate[id], ConfigMapUpdate{Key: key})
			} else {
				shouldGZIP := cfg.GZIP
				if cm.GZIP != nil {
					shouldGZIP = *cm.GZIP
				}
				toUpdate[id] = append(toUpdate[id], ConfigMapUpdate{Key: key, Filename: change.Path, GZIP: shouldGZIP})
			}
		}
	}
	return toUpdate
}

func handle(spc scmProviderClient, kc corev1.ConfigMapsGetter, cp commentPruner, defaultNamespace string, log *logrus.Entry, pre scm.PullRequestHook, config plugins.ConfigUpdater) error {
	// Only consider PRs with relevant actions
	isMerge := false
	isUpdate := false

	switch pre.Action {
	case scm.ActionClose, scm.ActionMerge:
		isMerge = true
	case scm.ActionOpen, scm.ActionReopen, scm.ActionSync:
		isUpdate = true
	case scm.ActionEdited, scm.ActionUpdate:
		changes := pre.Changes
		if changes.Base.Ref.From != "" || changes.Base.Sha.From != "" {
			isUpdate = true
		}
	}
	if !isMerge && !isUpdate {
		return nil
	}

	if len(config.Maps) == 0 { // Nothing to update
		return nil
	}

	pr := pre.PullRequest

	if isMerge && (!pr.Merged || pr.MergeSha == "" || pr.Base.Repo.Branch != pr.Base.Ref) {
		return nil
	}

	org := pr.Base.Repo.Namespace
	repo := pr.Base.Repo.Name

	// Which files changed in this PR?
	changes, err := spc.GetPullRequestChanges(org, repo, pr.Number)
	if err != nil {
		return err
	}

	// Are any of the changes files ones that define a configmap we want to update?
	toUpdate := FilterChanges(config, changes, log)

	if len(toUpdate) == 0 {
		return nil
	}

	if isMerge {
		message := func(name, namespace string, updates []ConfigMapUpdate, indent string) string {
			identifier := fmt.Sprintf("`%s` configmap", name)
			if namespace != "" {
				identifier = fmt.Sprintf("%s in namespace `%s`", identifier, namespace)
			}
			msg := fmt.Sprintf("%s using the following files:", identifier)
			for _, u := range updates {
				msg = fmt.Sprintf("%s\n%s- key `%s` using file `%s`", msg, indent, u.Key, u.Filename)
			}
			return msg
		}

		var updated []string
		indent := " " // one space
		if len(toUpdate) > 1 {
			indent = "   " // three spaces for sub bullets
		}
		for cm, data := range toUpdate {
			if cm.Namespace == "" {
				cm.Namespace = defaultNamespace
			}
			logger := log.WithFields(logrus.Fields{"configmap": map[string]string{"name": cm.Name, "namespace": cm.Namespace}})
			if err := Update(&scmFileGetter{org: org, repo: repo, commit: pr.MergeSha, client: spc}, kc.ConfigMaps(cm.Namespace), cm.Name, cm.Namespace, data, logger); err != nil {
				return err
			}
			updated = append(updated, message(cm.Name, cm.Namespace, data, indent))
		}

		var msg string
		switch n := len(updated); n {
		case 0:
			return nil
		case 1:
			msg = fmt.Sprintf("Updated the %s", updated[0])
		default:
			msg = fmt.Sprintf("Updated the following %d configmaps:\n", n)
			for _, updateMsg := range updated {
				msg += fmt.Sprintf(" * %s\n", updateMsg) // one space indent
			}
		}

		if err := spc.CreateComment(org, repo, pr.Number, true, plugins.FormatResponseRaw(pr.Body, pr.Link, pr.Author.Login, msg)); err != nil {
			return fmt.Errorf("comment err: %v", err)
		}
		return nil
	}

	var validationErrors []string

	baseURL, err := url.Parse(pr.Link)
	if err != nil {
		return errors2.Wrapf(err, "failed to parse URL %s", pr.Link)
	}
	baseURL.Path = ""
	baseURL.RawQuery = ""

	for cm, data := range toUpdate {
		fg := &scmFileGetter{
			org:    org,
			repo:   repo,
			commit: pr.Head.Sha,
			client: spc,
		}

		logger := log.WithFields(logrus.Fields{"configmap": map[string]string{"name": cm.Name, "namespace": cm.Namespace}})
		for _, upd := range data {
			content, err := fg.GetFile(upd.Filename)
			if err != nil {
				validationErrors = append(validationErrors, fmt.Sprintf("reading file %s: %s", upd.Filename, err.Error()))
				break
			}
			logger.WithFields(logrus.Fields{"key": upd.Key, "cmName": upd.Filename, "configmap": cm.Name}).Debug("Validating data.")

			var yamlErr error
			switch cm.Name {
			case "config":
				_, yamlErr = config2.LoadYAMLConfig(content)
			case "plugins":
				ca := &plugins.ConfigAgent{}
				_, yamlErr = ca.LoadYAMLConfig(content)
			default:
				m := make(map[interface{}]interface{})
				yamlErr = yaml.Unmarshal(content, &m)
			}
			if yamlErr != nil {
				link := util.BlobURLForProvider(spc.ProviderType(), baseURL, org, repo, pr.Head.Sha, upd.Filename)
				validationErrors = append(validationErrors, fmt.Sprintf("In file [%s](%s) for config map **%s**:\n%s", upd.Filename, link, cm.Name, indentErrMsg(yamlErr)))
			}
		}
	}

	cp.PruneComments(true, func(comment *scm.Comment) bool {
		return strings.Contains(comment.Body, configUpdaterMsgPruneMatch)
	})

	var statusInput *scm.StatusInput
	message := ""

	if len(validationErrors) > 0 {
		statusInput = &scm.StatusInput{
			State: scm.StateFailure,
			Label: configUpdaterContextName,
			Desc:  configUpdaterContextMsgFailed,
		}
		message = fmt.Sprintf("%s\n\n%s", configUpdaterMsgPruneMatch, strings.Join(validationErrors, "\n\n---\n\n"))
	} else {
		statusInput = &scm.StatusInput{
			State: scm.StateSuccess,
			Label: configUpdaterContextName,
			Desc:  configUpdaterContextMsgSuccess,
		}
	}

	if _, err := spc.CreateStatus(org, repo, pr.Head.Sha, statusInput); err != nil {
		resp := fmt.Sprintf("Cannot update PR status for context %s", statusInput.Label)
		log.WithError(err).Warn(resp)
	}
	if message != "" {
		return spc.CreateComment(org, repo, pr.Number, true, message)
	}
	return nil
}

func indentErrMsg(err error) string {
	var lines []string
	sc := bufio.NewScanner(strings.NewReader(err.Error()))
	for sc.Scan() {
		lines = append(lines, "> "+sc.Text())
	}
	return strings.Join(lines, "\n")
}
