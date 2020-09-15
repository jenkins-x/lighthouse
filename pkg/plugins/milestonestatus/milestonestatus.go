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

// Package milestonestatus implements the `/status` command which allows members of the milestone
// maintainers team to specify a `status/*` label to be applied to an Issue or PR.
package milestonestatus

import (
	"fmt"
	"strings"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/sirupsen/logrus"

	"github.com/jenkins-x/lighthouse/pkg/plugins"
)

const pluginName = "milestonestatus"

var (
	mustBeAuthorized = "You must be a member of the [%s/%s](https://github.com/orgs/%s/teams/%s/members) GitHub team to add status labels. If you believe you should be able to issue the /status command, please contact your %s and have them propose you as an additional delegate for this responsibility."
	milestoneTeamMsg = "The milestone maintainers team is the GitHub team %q with ID: %d."
	statusMap        = map[string]string{
		"approved-for-milestone": "status/approved-for-milestone",
		"in-progress":            "status/in-progress",
		"in-review":              "status/in-review",
	}
)

var (
	plugin = plugins.Plugin{
		Description:        "The milestonestatus plugin allows members of the milestone maintainers GitHub team to specify the 'status/*' label that should apply to a pull request.",
		ConfigHelpProvider: configHelp,
		Commands: []plugins.Command{{
			Name: "status",
			Arg: &plugins.CommandArg{
				Pattern: ".+",
			},
			Description: "Applies the 'status/' label to a PR.",
			WhoCanUse:   "Members of the milestone maintainers GitHub team can use the '/status' command. This team is specified in the config by providing the GitHub team's ID.",
			Action: plugins.
				Invoke(func(match plugins.CommandMatch, pc plugins.Agent, e scmprovider.GenericCommentEvent) error {
					return handle(match.Arg, pc.SCMProviderClient, pc.Logger, &e, pc.PluginConfig.RepoMilestone)
				}).
				When(plugins.Action(scm.ActionCreate)),
		}},
	}
)

type scmProviderClient interface {
	CreateComment(owner, repo string, number int, pr bool, comment string) error
	AddLabel(owner, repo string, number int, label string, pr bool) error
	ListTeamMembers(id int, role string) ([]*scm.TeamMember, error)
}

func init() {
	plugins.RegisterPlugin(pluginName, plugin)
}

func configHelp(config *plugins.Configuration, enabledRepos []string) (map[string]string, error) {
	msgForTeam := func(team plugins.Milestone) string {
		return fmt.Sprintf(milestoneTeamMsg, team.MaintainersTeam, team.MaintainersID)
	}
	configMap := make(map[string]string)
	for _, repo := range enabledRepos {
		team, exists := config.RepoMilestone[repo]
		if exists {
			configMap[repo] = msgForTeam(team)
		}
	}
	configMap[""] = msgForTeam(config.RepoMilestone[""])
	return configMap, nil
}

func handle(mileStone string, spc scmProviderClient, log *logrus.Entry, e *scmprovider.GenericCommentEvent, repoMilestone map[string]plugins.Milestone) error {
	org := e.Repo.Namespace
	repo := e.Repo.Name

	milestone, exists := repoMilestone[fmt.Sprintf("%s/%s", org, repo)]
	if !exists {
		// fallback default
		milestone = repoMilestone[""]
	}

	milestoneMaintainers, err := spc.ListTeamMembers(milestone.MaintainersID, scmprovider.RoleAll)
	if err != nil {
		return err
	}
	found := false
	for _, person := range milestoneMaintainers {
		login := strings.ToLower(e.Author.Login)
		if strings.ToLower(person.Login) == login {
			found = true
			break
		}
	}
	if !found {
		// not in the milestone maintainers team
		msg := fmt.Sprintf(mustBeAuthorized, org, milestone.MaintainersTeam, org, milestone.MaintainersTeam, milestone.MaintainersFriendlyName)
		return spc.CreateComment(org, repo, e.Number, e.IsPR, msg)
	}

	sLabel, validStatus := statusMap[strings.TrimSpace(mileStone)]
	if validStatus {
		if err := spc.AddLabel(org, repo, e.Number, sLabel, e.IsPR); err != nil {
			log.WithError(err).Errorf("Error adding the label %q to %s/%s#%d.", sLabel, org, repo, e.Number)
		}
	}

	return nil
}
