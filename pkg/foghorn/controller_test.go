package foghorn

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	lighthousev1alpha1 "github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/jenkins-x/lighthouse/pkg/watcher"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

func TestReconcile(t *testing.T) {
	testCases := []string{
		"status-change",
		"no-status-change",
	}

	oldToken := os.Getenv("GIT_TOKEN")
	err := os.Setenv("GIT_TOKEN", "abcd")
	assert.NoError(t, err)
	defer func() {
		if oldToken != "" {
			os.Setenv("GIT_TOKEN", oldToken)
		} else {
			os.Unsetenv("GIT_TOKEN")
		}
	}()
	configAgent := &config.Agent{}
	configAgent.Set(&config.Config{
		JobConfig: config.JobConfig{},
		ProwConfig: config.ProwConfig{
			Keeper:                 config.Keeper{},
			Plank:                  config.Plank{},
			BranchProtection:       config.BranchProtection{},
			Orgs:                   nil,
			JenkinsOperators:       nil,
			LighthouseJobNamespace: "",
			PodNamespace:           "",
			LogLevel:               "",
			PushGateway:            config.PushGateway{},
			OwnersDirExcludes:      nil,
			OwnersDirBlacklist:     nil,
			PubSubSubscriptions:    nil,
			GitHubOptions:          config.GitHubOptions{},
			ProviderConfig: &config.ProviderConfig{
				Kind:    "fake",
				Server:  "https://github.com",
				BotUser: "jenkins-x-bot",
			},
		},
	})

	pluginAgent := &plugins.ConfigAgent{}
	pluginAgent.Set(&plugins.Configuration{
		Plugins:              nil,
		ExternalPlugins:      nil,
		Owners:               plugins.Owners{},
		Approve:              nil,
		Blockades:            nil,
		Cat:                  plugins.Cat{},
		CherryPickUnapproved: plugins.CherryPickUnapproved{},
		ConfigUpdater:        plugins.ConfigUpdater{},
		Heart:                plugins.Heart{},
		Label:                plugins.Label{},
		Lgtm:                 nil,
		RepoMilestone:        nil,
		RequireMatchingLabel: nil,
		RequireSIG:           plugins.RequireSIG{},
		SigMention:           plugins.SigMention{},
		Size:                 plugins.Size{},
		Triggers:             nil,
		Welcome:              nil,
	})

	cfgMapWatcher := &watcher.ConfigMapWatcher{}

	ns := "jx"

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			testData := path.Join("test_data", tc)
			_, err := os.Stat(testData)
			assert.NoError(t, err)

			observedJob, err := loadLighthouseJob(testData, "observed-lhjob.yml")
			assert.NoError(t, err)

			expectedJob, err := loadLighthouseJob(testData, "expected-lhjob.yml")
			assert.NoError(t, err)

			// create fake controller
			scheme := runtime.NewScheme()
			err = lighthousev1alpha1.AddToScheme(scheme)
			assert.NoError(t, err)
			c := fake.NewFakeClientWithScheme(scheme, observedJob)
			reconciler, err := NewLighthouseJobReconcilerWithConfig(c, scheme, ns, cfgMapWatcher, configAgent, pluginAgent)
			assert.NoError(t, err)

			// invoke reconcile
			_, err = reconciler.Reconcile(ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: ns,
					Name:      observedJob.GetName(),
				},
			})
			assert.NoError(t, err)

			var jobList lighthousev1alpha1.LighthouseJobList
			err = c.List(nil, &jobList, client.InNamespace(ns))
			assert.NoError(t, err)
			assert.Len(t, jobList.Items, 1)
			// Ignore status.starttime since that's always going to be different
			updatedJob := jobList.Items[0].DeepCopy()
			updatedJob.Status.StartTime = metav1.Time{}
			if d := cmp.Diff(expectedJob.Status, updatedJob.Status); d != "" {
				t.Errorf("LighthouseJob did not match expected: %s", d)
			}

		})
	}
}

func loadLighthouseJob(dir string, baseFn string) (*lighthousev1alpha1.LighthouseJob, error) {
	fileName := filepath.Join(dir, baseFn)
	exists, err := util.FileExists(fileName)
	if err != nil {
		return nil, err
	}
	if exists {
		lhjob := &lighthousev1alpha1.LighthouseJob{}
		data, err := ioutil.ReadFile(fileName)
		if err != nil {
			return nil, err
		}
		err = yaml.Unmarshal(data, lhjob)
		if err != nil {
			return nil, err
		}
		return lhjob, err
	}
	return nil, nil
}
