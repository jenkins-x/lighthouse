package plugins_test

import (
	"reflect"
	"testing"

	"github.com/jenkins-x/lighthouse/pkg/pluginhelp"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
)

func TestCommandArgGetRegex(t *testing.T) {
	cases := []struct {
		name       string
		commandArg plugins.CommandArg
		expected   string
	}{
		{
			name: "optional",
			commandArg: plugins.CommandArg{
				Pattern:  "foo|bar",
				Optional: true,
			},
			expected: `(?:[ \t]+(foo|bar))?`,
		},
		{
			name: "not optional",
			commandArg: plugins.CommandArg{
				Pattern: "foo|bar",
			},
			expected: `(?:[ \t]+(foo|bar))`,
		},
		{
			name: "no pattern means everything",
			commandArg: plugins.CommandArg{
				Optional: true,
			},
			expected: `(?:[ \t]+([^\r\n]+))?`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.commandArg.GetRegex()
			if actual != tc.expected {
				t.Errorf("actual regex does not match expected %s != expected %s", actual, tc.expected)
			}
		})
	}
}

func TestCommandArgGetUsage(t *testing.T) {
	cases := []struct {
		name       string
		commandArg plugins.CommandArg
		expected   string
	}{
		{
			name: "optional with no usage",
			commandArg: plugins.CommandArg{
				Pattern:  "foo|bar",
				Optional: true,
			},
			expected: "[foo|bar]",
		},
		{
			name: "not optional with no usage",
			commandArg: plugins.CommandArg{
				Pattern: "foo|bar",
			},
			expected: "<foo|bar>",
		},
		{
			name: "optional no pattern",
			commandArg: plugins.CommandArg{
				Optional: true,
			},
			expected: "[anything]",
		},
		{
			name:       "not optional no pattern",
			commandArg: plugins.CommandArg{},
			expected:   "<anything>",
		},
		{
			name: "optional with usage",
			commandArg: plugins.CommandArg{
				Pattern:  "foo|bar",
				Usage:    "option_name",
				Optional: true,
			},
			expected: "[option_name]",
		},
		{
			name: "not optional with usage",
			commandArg: plugins.CommandArg{
				Pattern: "foo|bar",
				Usage:   "option_name",
			},
			expected: "<option_name>",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.commandArg.GetUsage()
			if actual != tc.expected {
				t.Errorf("actual usage does not match expected %s != expected %s", actual, tc.expected)
			}
		})
	}
}

func TestCommandGetRegex(t *testing.T) {
	cases := []struct {
		name     string
		command  plugins.Command
		expected string
	}{
		{
			name: "no prefix and no arg",
			command: plugins.Command{
				Name: "test",
			},
			expected: `(?mi)^/(?:lh-)?(test)\s*$`,
		},
		{
			name: "prefix and no arg",
			command: plugins.Command{
				Prefix: "prefix",
				Name:   "test",
			},
			expected: `(?mi)^/(?:lh-)?(prefix)?(test)\s*$`,
		},
		{
			name: "prefix and optional arg",
			command: plugins.Command{
				Prefix: "prefix",
				Name:   "test",
				Arg: &plugins.CommandArg{
					Pattern:  "foo|bar",
					Optional: true,
				},
			},
			expected: `(?mi)^/(?:lh-)?(prefix)?(test)(?:[ \t]+(foo|bar))?\s*$`,
		},
		{
			name: "prefix and arg",
			command: plugins.Command{
				Prefix: "prefix",
				Name:   "test",
				Arg: &plugins.CommandArg{
					Pattern: "foo|bar",
				},
			},
			expected: `(?mi)^/(?:lh-)?(prefix)?(test)(?:[ \t]+(foo|bar))\s*$`,
		},
		{
			name: "no prefix and optional arg",
			command: plugins.Command{
				Name: "test",
				Arg: &plugins.CommandArg{
					Pattern:  "foo|bar",
					Optional: true,
				},
			},
			expected: `(?mi)^/(?:lh-)?(test)(?:[ \t]+(foo|bar))?\s*$`,
		},
		{
			name: "no prefix and arg",
			command: plugins.Command{
				Name: "test",
				Arg: &plugins.CommandArg{
					Pattern: "foo|bar",
				},
			},
			expected: `(?mi)^/(?:lh-)?(test)(?:[ \t]+(foo|bar))\s*$`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.command.GetRegex()
			if actual == nil {
				t.Errorf("actual regex is nil")
			} else {
				if actual.String() != tc.expected {
					t.Errorf("actual usage does not match expected %s != expected %s", actual, tc.expected)
				}
			}
		})
	}
}

func TestCommandGetRegexCached(t *testing.T) {
	command := plugins.Command{
		Prefix: "prefix",
		Name:   "test",
		Arg: &plugins.CommandArg{
			Pattern: "foo|bar",
		},
	}
	re1 := command.GetRegex()
	re2 := command.GetRegex()
	if re1 != re2 {
		t.Errorf("command regex should have been cached")
	}
}

func TestCommandGetMatches(t *testing.T) {
	cases := []struct {
		name     string
		command  plugins.Command
		content  string
		expected []plugins.CommandMatch
	}{
		{
			name: "no prefix and no arg, content match",
			command: plugins.Command{
				Name: "test",
			},
			content: "/test",
			expected: []plugins.CommandMatch{{
				Name: "test",
			}},
		},
		{
			name: "no prefix and no arg, content not match arg",
			command: plugins.Command{
				Name: "test",
			},
			content: "/test test",
		},
		{
			name: "no prefix and no arg, content not match prefix",
			command: plugins.Command{
				Name: "test",
			},
			content: "/re-test",
		},
		{
			name: "no prefix and no arg, content not match command",
			command: plugins.Command{
				Name: "test",
			},
			content: "/build",
		},
		{
			name: "prefix and no arg, content match",
			command: plugins.Command{
				Prefix: "prefix-",
				Name:   "test",
			},
			content: "/prefix-test",
			expected: []plugins.CommandMatch{{
				Prefix: "prefix-",
				Name:   "test",
			}},
		},
		{
			name: "prefix and no arg, content not match arg",
			command: plugins.Command{
				Prefix: "prefix-",
				Name:   "test",
			},
			content: "/prefix-test test",
		},
		{
			name: "prefix and no arg, content not match prefix",
			command: plugins.Command{
				Prefix: "prefix-",
				Name:   "test",
			},
			content: "/wrong-test",
		},
		{
			name: "prefix and no arg, content not match command",
			command: plugins.Command{
				Prefix: "prefix-",
				Name:   "test",
			},
			content: "/build",
		},
		{
			name: "no prefix and arg, content match",
			command: plugins.Command{
				Name: "test",
				Arg: &plugins.CommandArg{
					Pattern: "foo|bar",
				},
			},
			content: "/test foo",
			expected: []plugins.CommandMatch{{
				Name: "test",
				Arg:  "foo",
			}},
		},
		{
			name: "no prefix and arg, content not match arg",
			command: plugins.Command{
				Name: "test",
				Arg: &plugins.CommandArg{
					Pattern: "foo",
				},
			},
			content: "/test bar",
		},
		{
			name: "no prefix and arg, content not match prefix",
			command: plugins.Command{
				Name: "test",
				Arg: &plugins.CommandArg{
					Pattern: "foo",
				},
			},
			content: "/wrong-test foo",
		},
		{
			name: "no prefix and arg, content not match command",
			command: plugins.Command{
				Name: "test",
				Arg: &plugins.CommandArg{
					Pattern: "foo",
				},
			},
			content: "/build foo",
		},
		{
			name: "prefix and arg, content match with prefix",
			command: plugins.Command{
				Prefix: "prefix-",
				Name:   "test",
				Arg: &plugins.CommandArg{
					Pattern: "foo|bar",
				},
			},
			content: "/prefix-test foo",
			expected: []plugins.CommandMatch{{
				Prefix: "prefix-",
				Name:   "test",
				Arg:    "foo",
			}},
		},
		{
			name: "prefix and arg, content match without prefix",
			command: plugins.Command{
				Prefix: "prefix-",
				Name:   "test",
				Arg: &plugins.CommandArg{
					Pattern: "foo|bar",
				},
			},
			content: "/test foo",
			expected: []plugins.CommandMatch{{
				Name: "test",
				Arg:  "foo",
			}},
		},
		{
			name: "prefix and arg, content match without arg",
			command: plugins.Command{
				Prefix: "prefix-",
				Name:   "test",
				Arg: &plugins.CommandArg{
					Pattern:  "foo|bar",
					Optional: true,
				},
			},
			content: "/test",
			expected: []plugins.CommandMatch{{
				Name: "test",
			}},
		},
		{
			name: "prefix and arg, content match with prefix and without arg",
			command: plugins.Command{
				Prefix: "prefix-",
				Name:   "test",
				Arg: &plugins.CommandArg{
					Pattern:  "foo|bar",
					Optional: true,
				},
			},
			content: "/prefix-test",
			expected: []plugins.CommandMatch{{
				Prefix: "prefix-",
				Name:   "test",
			}},
		},
		{
			name: "prefix and arg, content not match arg",
			command: plugins.Command{
				Prefix: "prefix-",
				Name:   "test",
				Arg: &plugins.CommandArg{
					Pattern: "foo",
				},
			},
			content: "/test bar",
		},
		{
			name: "prefix and arg, content not match prefix",
			command: plugins.Command{
				Prefix: "prefix-",
				Name:   "test",
				Arg: &plugins.CommandArg{
					Pattern: "foo",
				},
			},
			content: "/wrong-test foo",
		},
		{
			name: "prefix and arg, content not match command",
			command: plugins.Command{
				Prefix: "prefix-",
				Name:   "test",
				Arg: &plugins.CommandArg{
					Pattern: "foo",
				},
			},
			content: "/build foo",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			matches, err := tc.command.GetMatches(tc.content)
			if err != nil {
				t.Errorf("an error has occured %w", err)
			} else {
				if !reflect.DeepEqual(tc.expected, matches) {
					t.Errorf("expected matches %q, but got %q", tc.expected, matches)
				}
			}
		})
	}
}

func TestCommandGetHelp(t *testing.T) {
	cases := []struct {
		name     string
		command  plugins.Command
		expected pluginhelp.Command
	}{
		{
			name: "no prefix and no arg",
			command: plugins.Command{
				Name:        "test",
				Description: "Some command description",
			},
			expected: pluginhelp.Command{
				Usage:       "/[lh-]test",
				Featured:    false,
				Description: "Some command description",
				WhoCanUse:   "Anyone",
				Examples: []string{
					"/test",
					"/lh-test",
				},
			},
		},
		{
			name: "prefix and no arg",
			command: plugins.Command{
				Prefix:      "prefix-",
				Name:        "test",
				Description: "Some command description",
			},
			expected: pluginhelp.Command{
				Usage:       "/[lh-][prefix-]test",
				Featured:    false,
				Description: "Some command description",
				WhoCanUse:   "Anyone",
				Examples: []string{
					"/test",
					"/lh-test",
					"/prefix-test",
					"/lh-prefix-test",
				},
			},
		},
		{
			name: "prefix and optional arg",
			command: plugins.Command{
				Prefix: "prefix-",
				Name:   "test",
				Arg: &plugins.CommandArg{
					Pattern:  "foo|bar",
					Optional: true,
				},
				Description: "Some command description",
			},
			expected: pluginhelp.Command{
				Usage:       "/[lh-][prefix-]test [foo|bar]",
				Featured:    false,
				Description: "Some command description",
				WhoCanUse:   "Anyone",
				Examples: []string{
					"/test",
					"/lh-test",
					"/prefix-test",
					"/lh-prefix-test",
				},
			},
		},
		{
			name: "prefix and optional arg no pattern",
			command: plugins.Command{
				Prefix: "prefix-",
				Name:   "test",
				Arg: &plugins.CommandArg{
					Optional: true,
				},
				Description: "Some command description",
			},
			expected: pluginhelp.Command{
				Usage:       "/[lh-][prefix-]test [anything]",
				Featured:    false,
				Description: "Some command description",
				WhoCanUse:   "Anyone",
				Examples: []string{
					"/test",
					"/lh-test",
					"/prefix-test",
					"/lh-prefix-test",
				},
			},
		},
		{
			name: "prefix and optional arg with usage",
			command: plugins.Command{
				Prefix: "prefix-",
				Name:   "test",
				Arg: &plugins.CommandArg{
					Usage:    "arg description",
					Optional: true,
				},
				Description: "Some command description",
			},
			expected: pluginhelp.Command{
				Usage:       "/[lh-][prefix-]test [arg description]",
				Featured:    false,
				Description: "Some command description",
				WhoCanUse:   "Anyone",
				Examples: []string{
					"/test",
					"/lh-test",
					"/prefix-test",
					"/lh-prefix-test",
				},
			},
		},
		{
			name: "prefix and arg",
			command: plugins.Command{
				Prefix: "prefix-",
				Name:   "test",
				Arg: &plugins.CommandArg{
					Pattern: "foo|bar",
				},
				Description: "Some command description",
			},
			expected: pluginhelp.Command{
				Usage:       "/[lh-][prefix-]test <foo|bar>",
				Featured:    false,
				Description: "Some command description",
				WhoCanUse:   "Anyone",
				Examples: []string{
					"/test",
					"/lh-test",
					"/prefix-test",
					"/lh-prefix-test",
				},
			},
		},
		{
			name: "prefix and arg no pattern",
			command: plugins.Command{
				Prefix:      "prefix-",
				Name:        "test",
				Arg:         &plugins.CommandArg{},
				Description: "Some command description",
			},
			expected: pluginhelp.Command{
				Usage:       "/[lh-][prefix-]test <anything>",
				Featured:    false,
				Description: "Some command description",
				WhoCanUse:   "Anyone",
				Examples: []string{
					"/test",
					"/lh-test",
					"/prefix-test",
					"/lh-prefix-test",
				},
			},
		},
		{
			name: "prefix and arg with usage",
			command: plugins.Command{
				Prefix: "prefix-",
				Name:   "test",
				Arg: &plugins.CommandArg{
					Usage: "arg description",
				},
				Description: "Some command description",
			},
			expected: pluginhelp.Command{
				Usage:       "/[lh-][prefix-]test <arg description>",
				Featured:    false,
				Description: "Some command description",
				WhoCanUse:   "Anyone",
				Examples: []string{
					"/test",
					"/lh-test",
					"/prefix-test",
					"/lh-prefix-test",
				},
			},
		},
		{
			name: "featured",
			command: plugins.Command{
				Prefix: "prefix-",
				Name:   "test",
				Arg: &plugins.CommandArg{
					Pattern: "foo|bar",
				},
				Description: "Some command description",
				Featured:    true,
			},
			expected: pluginhelp.Command{
				Usage:       "/[lh-][prefix-]test <foo|bar>",
				Featured:    true,
				Description: "Some command description",
				WhoCanUse:   "Anyone",
				Examples: []string{
					"/test",
					"/lh-test",
					"/prefix-test",
					"/lh-prefix-test",
				},
			},
		},
		{
			name: "who can use",
			command: plugins.Command{
				Prefix: "prefix-",
				Name:   "test",
				Arg: &plugins.CommandArg{
					Pattern: "foo|bar",
				},
				Description: "Some command description",
				WhoCanUse:   "only admins",
			},
			expected: pluginhelp.Command{
				Usage:       "/[lh-][prefix-]test <foo|bar>",
				Description: "Some command description",
				WhoCanUse:   "only admins",
				Examples: []string{
					"/test",
					"/lh-test",
					"/prefix-test",
					"/lh-prefix-test",
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			help := tc.command.GetHelp()
			if !reflect.DeepEqual(tc.expected, help) {
				t.Errorf("expected help %v, but got %v", tc.expected, help)
			}
		})
	}
}
