package plugins_test

import (
	"reflect"
	"testing"

	"github.com/jenkins-x/lighthouse/pkg/pluginhelp"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"k8s.io/apimachinery/pkg/util/sets"
)

func TestPluginIsProviderExcluded(t *testing.T) {
	cases := []struct {
		name     string
		plugin   plugins.Plugin
		provider string
		expected bool
	}{
		{
			name:     "no exclusion",
			plugin:   plugins.Plugin{},
			provider: "foo",
			expected: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.plugin.IsProviderExcluded(tc.provider)
			if actual != tc.expected {
				t.Errorf("provider exclusion check for provider %s does not match expected %t != expected %t", tc.provider, actual, tc.expected)
			}
		})
	}
}

func TestPluginGetHelp(t *testing.T) {
	cases := []struct {
		name         string
		plugin       plugins.Plugin
		config       *plugins.Configuration
		enabledRepos []string
		expected     *pluginhelp.PluginHelp
		shouldError  bool
	}{
		{
			name: "no config help provider",
			plugin: plugins.Plugin{
				Description:           "Some plugin description",
				ExcludedProviders:     sets.NewString(),
				ConfigHelpProvider:    nil,
				IssueHandler:          nil,
				PullRequestHandler:    nil,
				PushEventHandler:      nil,
				ReviewEventHandler:    nil,
				StatusEventHandler:    nil,
				GenericCommentHandler: nil,
			},
			config:       nil,
			enabledRepos: nil,
			expected: &pluginhelp.PluginHelp{
				Description:       "Some plugin description",
				ExcludedProviders: []string{},
			},
		},
		{
			name: "excluded providers",
			plugin: plugins.Plugin{
				Description:           "Some plugin description",
				ExcludedProviders:     sets.NewString("foo"),
				ConfigHelpProvider:    nil,
				IssueHandler:          nil,
				PullRequestHandler:    nil,
				PushEventHandler:      nil,
				ReviewEventHandler:    nil,
				StatusEventHandler:    nil,
				GenericCommentHandler: nil,
			},
			config:       nil,
			enabledRepos: nil,
			expected: &pluginhelp.PluginHelp{
				Description: "Some plugin description",
				ExcludedProviders: []string{
					"foo",
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := tc.plugin.GetHelp(tc.config, tc.enabledRepos)
			if !tc.shouldError && err != nil {
				t.Errorf("%s: didn't expect error: %v", tc.name, err)
			} else if tc.shouldError && err == nil {
				t.Errorf("%s: expected an error to occur", tc.name)
			} else {
				if !reflect.DeepEqual(tc.expected, actual) {
					t.Errorf("plugin help does not match expected %+v != expected %+v", actual, tc.expected)
				}
			}
		})
	}
}
