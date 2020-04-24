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

// Package milestone implements the `/milestone` command which allows members of the milestone
// maintainers team to specify a milestone to be applied to an Issue or PR.
package milestone

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/sirupsen/logrus"

	"github.com/jenkins-x/lighthouse/pkg/pluginhelp"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
)

const pluginName = "milestone"

var (
	milestoneRegex   = regexp.MustCompile(`(?m)^/(?:lh-)?milestone\s+(.+?)\s*$`)
	mustBeAuthorized = "You must be a member of the [%s/%s](https://github.com/orgs/%s/teams/%s/members) GitHub team to set the milestone. If you believe you should be able to issue the /milestone command, please contact your %s and have them propose you as an additional delegate for this responsibility."
	invalidMilestone = "The provided milestone is not valid for this repository. Milestones in this repository: [%s]\n\nUse `/milestone %s` to clear the milestone."
	milestoneTeamMsg = "The milestone maintainers team is the GitHub team %q with ID: %d."
	clearKeyword     = "clear"
)

type scmProviderClient interface {
	CreateComment(owner, repo string, number int, pr bool, comment string) error
	ClearMilestone(org, repo string, num int) error
	SetMilestone(org, repo string, issueNum, milestoneNum int) error
	ListTeamMembers(id int, role string) ([]*scm.TeamMember, error)
	ListMilestones(org, repo string) ([]scmprovider.Milestone, error)
}

func init() {
	plugins.RegisterGenericCommentHandler(pluginName, handleGenericComment, helpProvider)
}

func helpProvider(config *plugins.Configuration, enabledRepos []string) (*pluginhelp.PluginHelp, error) {
	msgForTeam := func(team plugins.Milestone) string {
		return fmt.Sprintf(milestoneTeamMsg, team.MaintainersTeam, team.MaintainersID)
	}

	pluginHelp := &pluginhelp.PluginHelp{
		Description: "The milestone plugin allows members of a configurable GitHub team to set the milestone on an issue or pull request.",
		Config: func(repos []string) map[string]string {
			configMap := make(map[string]string)
			for _, repo := range repos {
				team, exists := config.RepoMilestone[repo]
				if exists {
					configMap[repo] = msgForTeam(team)
				}
			}
			configMap[""] = msgForTeam(config.RepoMilestone[""])
			return configMap
		}(enabledRepos),
	}
	pluginHelp.AddCommand(pluginhelp.Command{
		Usage:       "/milestone <version> or /milestone clear",
		Description: "Updates the milestone for an issue or PR",
		Featured:    false,
		WhoCanUse:   "Members of the milestone maintainers GitHub team can use the '/milestone' command.",
		Examples:    []string{"/milestone v1.10", "/milestone v1.9", "/milestone clear", "/lh-milestone clear"},
	})
	return pluginHelp, nil
}

func handleGenericComment(pc plugins.Agent, e scmprovider.GenericCommentEvent) error {
	return handle(pc.SCMProviderClient, pc.Logger, &e, pc.PluginConfig.RepoMilestone)
}

func buildMilestoneMap(milestones []scmprovider.Milestone) map[string]int {
	m := make(map[string]int)
	for _, ms := range milestones {
		m[ms.Title] = ms.Number
	}
	return m
}
func handle(spc scmProviderClient, log *logrus.Entry, e *scmprovider.GenericCommentEvent, repoMilestone map[string]plugins.Milestone) error {
	if e.Action != scm.ActionCreate {
		return nil
	}

	milestoneMatch := milestoneRegex.FindStringSubmatch(e.Body)
	if len(milestoneMatch) != 2 {
		return nil
	}

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
		login := scmprovider.NormLogin(e.Author.Login)
		if scmprovider.NormLogin(person.Login) == login {
			found = true
			break
		}
	}
	if !found {
		// not in the milestone maintainers team
		msg := fmt.Sprintf(mustBeAuthorized, org, milestone.MaintainersTeam, org, milestone.MaintainersTeam, milestone.MaintainersFriendlyName)
		return spc.CreateComment(org, repo, e.Number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, e.Author.Login, msg))
	}

	milestones, err := spc.ListMilestones(org, repo)
	if err != nil {
		log.WithError(err).Errorf("Error listing the milestones in the %s/%s repo", org, repo)
		return err
	}
	proposedMilestone := milestoneMatch[1]

	// special case, if the clear keyword is used
	if proposedMilestone == clearKeyword {
		if err := spc.ClearMilestone(org, repo, e.Number); err != nil {
			log.WithError(err).Errorf("Error clearing the milestone for %s/%s#%d.", org, repo, e.Number)
		}
		return nil
	}

	milestoneMap := buildMilestoneMap(milestones)
	milestoneNumber, ok := milestoneMap[proposedMilestone]
	if !ok {
		slice := make([]string, 0, len(milestoneMap))
		for k := range milestoneMap {
			slice = append(slice, fmt.Sprintf("`%s`", k))
		}
		sort.Strings(slice)

		msg := fmt.Sprintf(invalidMilestone, strings.Join(slice, ", "), clearKeyword)
		return spc.CreateComment(org, repo, e.Number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, e.Author.Login, msg))
	}

	if err := spc.SetMilestone(org, repo, e.Number, milestoneNumber); err != nil {
		log.WithError(err).Errorf("Error adding the milestone %s to %s/%s#%d.", proposedMilestone, org, repo, e.Number)
	}

	return nil
}
