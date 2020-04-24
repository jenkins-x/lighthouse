/*
Copyright 2016 The Kubernetes Authors.

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

package fake

import (
	"fmt"
	"regexp"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"k8s.io/apimachinery/pkg/util/sets"
)

const botName = "k8s-ci-robot"

const (
	// Bot is the exported botName
	Bot = botName
	// TestRef is the ref returned when calling GetRef
	TestRef = "abcde"
	// providerType is the fake provider name
	providerType = "fake"
)

// SCMClient is like client, but fake.
type SCMClient struct {
	Issues              map[int][]*scm.Issue
	OrgMembers          map[string][]string
	Collaborators       []string
	IssueComments       map[int][]*scm.Comment
	IssueCommentID      int
	PullRequests        map[int]*scm.PullRequest
	PullRequestChanges  map[int][]*scm.Change
	PullRequestComments map[int][]*scm.Comment
	ReviewID            int
	Reviews             map[int][]*scm.Review
	CombinedStatuses    map[string]*scm.CombinedStatus
	CreatedStatuses     map[string][]*scm.StatusInput
	IssueEvents         map[int][]*scm.ListedIssueEvent
	Commits             map[string]*scm.Commit

	//All Labels That Exist In The Repo
	RepoLabelsExisting []string
	// org/repo#number:label
	IssueLabelsAdded          []string
	IssueLabelsExisting       []string
	IssueLabelsRemoved        []string
	PullRequestLabelsAdded    []string
	PullRequestLabelsExisting []string
	PullRequestLabelsRemoved  []string

	// org/repo#number:body
	IssueCommentsAdded       []string
	PullRequestCommentsAdded []string
	// org/repo#issuecommentid
	IssueCommentsDeleted       []string
	PullRequestCommentsDeleted []string

	// org/repo#issuecommentid:reaction
	IssueReactionsAdded   []string
	CommentReactionsAdded []string

	// org/repo#number:assignee
	AssigneesAdded []string

	// org/repo#number:milestone (represents the milestone for a specific issue)
	Milestone    int
	MilestoneMap map[string]int

	// list of commits for each PR
	// org/repo#number:[]commit
	CommitMap map[string][]scm.Commit

	// Fake remote git storage. File name are keys
	// and values map SHA to content
	RemoteFiles map[string]map[string]string

	// A list of refs that got deleted via DeleteRef
	RefsDeleted []struct{ Org, Repo, Ref string }
}

// ProviderType returns the provider type
func (f *SCMClient) ProviderType() string {
	return providerType
}

// SupportsGraphQL returns whether the provider supports GraphQL
func (f *SCMClient) SupportsGraphQL() bool {
	return false
}

// BotName returns authenticated login.
func (f *SCMClient) BotName() (string, error) {
	return botName, nil
}

// IsMember returns true if user is in org.
func (f *SCMClient) IsMember(org, user string) (bool, error) {
	for _, m := range f.OrgMembers[org] {
		if m == user {
			return true, nil
		}
	}
	return false, nil
}

// ListIssueComments returns comments.
func (f *SCMClient) ListIssueComments(owner, repo string, number int) ([]*scm.Comment, error) {
	return append([]*scm.Comment{}, f.IssueComments[number]...), nil
}

// ListPullRequestComments returns review comments.
func (f *SCMClient) ListPullRequestComments(owner, repo string, number int) ([]*scm.Comment, error) {
	return append([]*scm.Comment{}, f.PullRequestComments[number]...), nil
}

// ListReviews returns reviews.
func (f *SCMClient) ListReviews(owner, repo string, number int) ([]*scm.Review, error) {
	return append([]*scm.Review{}, f.Reviews[number]...), nil
}

// ListIssueEvents returns issue events
func (f *SCMClient) ListIssueEvents(owner, repo string, number int) ([]*scm.ListedIssueEvent, error) {
	return append([]*scm.ListedIssueEvent{}, f.IssueEvents[number]...), nil
}

// CreateComment adds a comment to a PR
func (f *SCMClient) CreateComment(owner, repo string, number int, pr bool, comment string) error {
	if pr {
		f.PullRequestCommentsAdded = append(f.PullRequestCommentsAdded, fmt.Sprintf("%s/%s#%d:%s", owner, repo, number, comment))
		f.PullRequestComments[number] = append(f.PullRequestComments[number], &scm.Comment{
			ID:     f.IssueCommentID,
			Body:   comment,
			Author: scm.User{Login: botName},
		})
	} else {
		f.IssueCommentsAdded = append(f.IssueCommentsAdded, fmt.Sprintf("%s/%s#%d:%s", owner, repo, number, comment))
		f.IssueComments[number] = append(f.IssueComments[number], &scm.Comment{
			ID:     f.IssueCommentID,
			Body:   comment,
			Author: scm.User{Login: botName},
		})
	}
	f.IssueCommentID++
	return nil
}

// CreateReview adds a review to a PR
func (f *SCMClient) CreateReview(org, repo string, number int, r scmprovider.DraftReview) error {
	f.Reviews[number] = append(f.Reviews[number], &scm.Review{
		ID:     f.ReviewID,
		Author: scm.User{Login: botName},
		Body:   r.Body,
	})
	f.ReviewID++
	return nil
}

// CreateCommentReaction adds emoji to a comment.
func (f *SCMClient) CreateCommentReaction(org, repo string, ID int, reaction string) error {
	f.CommentReactionsAdded = append(f.CommentReactionsAdded, fmt.Sprintf("%s/%s#%d:%s", org, repo, ID, reaction))
	return nil
}

// CreateIssueReaction adds an emoji to an issue.
func (f *SCMClient) CreateIssueReaction(org, repo string, ID int, reaction string) error {
	f.IssueReactionsAdded = append(f.IssueReactionsAdded, fmt.Sprintf("%s/%s#%d:%s", org, repo, ID, reaction))
	return nil
}

// DeleteComment deletes a comment.
func (f *SCMClient) DeleteComment(owner, repo string, number, ID int, pr bool) error {
	if pr {
		f.PullRequestCommentsDeleted = append(f.PullRequestCommentsDeleted, fmt.Sprintf("%s/%s#%d", owner, repo, ID))
		for num, ics := range f.PullRequestComments {
			for i, ic := range ics {
				if ic.ID == ID {
					f.PullRequestComments[num] = append(ics[:i], ics[i+1:]...)
					return nil
				}
			}
		}
		return fmt.Errorf("could not find pull request comment %d", ID)
	}

	f.IssueCommentsDeleted = append(f.IssueCommentsDeleted, fmt.Sprintf("%s/%s#%d", owner, repo, ID))
	for num, ics := range f.IssueComments {
		for i, ic := range ics {
			if ic.ID == ID {
				f.IssueComments[num] = append(ics[:i], ics[i+1:]...)
				return nil
			}
		}
	}
	return fmt.Errorf("could not find issue comment %d", ID)
}

// DeleteStaleComments deletes comments flagged by isStale.
func (f *SCMClient) DeleteStaleComments(org, repo string, number int, comments []*scm.Comment, pr bool, isStale func(*scm.Comment) bool) error {
	if comments == nil {
		if pr {
			comments, _ = f.ListPullRequestComments(org, repo, number)
		} else {
			comments, _ = f.ListIssueComments(org, repo, number)
		}
	}
	for _, comment := range comments {
		if isStale(comment) {
			if err := f.DeleteComment(org, repo, number, comment.ID, pr); err != nil {
				return fmt.Errorf("failed to delete stale comment with ID '%d'", comment.ID)
			}
		}
	}
	return nil
}

// GetPullRequest returns details about the PR.
func (f *SCMClient) GetPullRequest(owner, repo string, number int) (*scm.PullRequest, error) {
	val, exists := f.PullRequests[number]
	if !exists {
		return nil, fmt.Errorf("Pull request number %d does not exit", number)
	}
	return val, nil
}

// GetPullRequestChanges returns the file modifications in a PR.
func (f *SCMClient) GetPullRequestChanges(org, repo string, number int) ([]*scm.Change, error) {
	return f.PullRequestChanges[number], nil
}

// GetRef returns the hash of a ref.
func (f *SCMClient) GetRef(owner, repo, ref string) (string, error) {
	return TestRef, nil
}

// DeleteRef returns an error indicating if deletion of the given ref was successful
func (f *SCMClient) DeleteRef(owner, repo, ref string) error {
	f.RefsDeleted = append(f.RefsDeleted, struct{ Org, Repo, Ref string }{Org: owner, Repo: repo, Ref: ref})
	return nil
}

// GetSingleCommit returns a single commit.
func (f *SCMClient) GetSingleCommit(org, repo, SHA string) (*scm.Commit, error) {
	return f.Commits[SHA], nil
}

// CreateStatus adds a status context to a commit.
func (f *SCMClient) CreateStatus(owner, repo, SHA string, s *scm.StatusInput) (*scm.Status, error) {
	if f.CreatedStatuses == nil {
		f.CreatedStatuses = make(map[string][]*scm.StatusInput)
	}
	statuses := f.CreatedStatuses[SHA]
	var updated bool
	for i := range statuses {
		if statuses[i].Label == s.Label {
			statuses[i] = s
			updated = true
		}
	}
	if !updated {
		statuses = append(statuses, s)
	}
	f.CreatedStatuses[SHA] = statuses
	return nil, nil
}

// ListStatuses returns individual status contexts on a commit.
func (f *SCMClient) ListStatuses(org, repo, ref string) ([]*scm.Status, error) {
	return scm.ConvertStatusInputsToStatuses(f.CreatedStatuses[ref]), nil
}

// GetCombinedStatus returns the overall status for a commit.
func (f *SCMClient) GetCombinedStatus(owner, repo, ref string) (*scm.CombinedStatus, error) {
	return f.CombinedStatuses[ref], nil
}

// GetRepoLabels gets labels in a repo.
func (f *SCMClient) GetRepoLabels(owner, repo string) ([]*scm.Label, error) {
	la := []*scm.Label{}
	for _, l := range f.RepoLabelsExisting {
		la = append(la, &scm.Label{Name: l})
	}
	return la, nil
}

// GetIssueLabels gets labels on an issue
func (f *SCMClient) GetIssueLabels(owner, repo string, number int, pr bool) ([]*scm.Label, error) {
	re := regexp.MustCompile(fmt.Sprintf(`^%s/%s#%d:(.*)$`, owner, repo, number))
	la := []*scm.Label{}
	var allLabels sets.String
	if pr {
		allLabels = sets.NewString(f.PullRequestLabelsExisting...)
		allLabels.Insert(f.PullRequestLabelsAdded...)
		allLabels.Delete(f.PullRequestLabelsRemoved...)
	} else {
		allLabels = sets.NewString(f.IssueLabelsExisting...)
		allLabels.Insert(f.IssueLabelsAdded...)
		allLabels.Delete(f.IssueLabelsRemoved...)
	}
	for _, l := range allLabels.List() {
		groups := re.FindStringSubmatch(l)
		if groups != nil {
			la = append(la, &scm.Label{Name: groups[1]})
		}
	}
	return la, nil
}

// AddLabel adds a label
func (f *SCMClient) AddLabel(owner, repo string, number int, label string, pr bool) error {
	labelString := fmt.Sprintf("%s/%s#%d:%s", owner, repo, number, label)
	if pr {
		if sets.NewString(f.PullRequestLabelsAdded...).Has(labelString) {
			return fmt.Errorf("cannot add %v to %s/%s/#%d", label, owner, repo, number)
		}
		if f.RepoLabelsExisting == nil {
			f.PullRequestLabelsAdded = append(f.PullRequestLabelsAdded, labelString)
			return nil
		}
		for _, l := range f.RepoLabelsExisting {
			if label == l {
				f.PullRequestLabelsAdded = append(f.PullRequestLabelsAdded, labelString)
				return nil
			}
		}
	} else {
		if sets.NewString(f.IssueLabelsAdded...).Has(labelString) {
			return fmt.Errorf("cannot add %v to %s/%s/#%d", label, owner, repo, number)
		}
		if f.RepoLabelsExisting == nil {
			f.IssueLabelsAdded = append(f.IssueLabelsAdded, labelString)
			return nil
		}
		for _, l := range f.RepoLabelsExisting {
			if label == l {
				f.IssueLabelsAdded = append(f.IssueLabelsAdded, labelString)
				return nil
			}
		}
	}
	return fmt.Errorf("cannot add %v to %s/%s/#%d", label, owner, repo, number)
}

// RemoveLabel removes a label
func (f *SCMClient) RemoveLabel(owner, repo string, number int, label string, pr bool) error {
	labelString := fmt.Sprintf("%s/%s#%d:%s", owner, repo, number, label)
	if pr {
		if !sets.NewString(f.PullRequestLabelsRemoved...).Has(labelString) {
			f.PullRequestLabelsRemoved = append(f.PullRequestLabelsRemoved, labelString)
			return nil
		}
	} else {
		if !sets.NewString(f.IssueLabelsRemoved...).Has(labelString) {
			f.IssueLabelsRemoved = append(f.IssueLabelsRemoved, labelString)
			return nil
		}
	}
	return fmt.Errorf("cannot remove %v from %s/%s/#%d", label, owner, repo, number)
}

// FindIssues returns f.Issues
func (f *SCMClient) FindIssues(query, sort string, asc bool) ([]scm.Issue, error) {
	var issues []scm.Issue
	for _, slice := range f.Issues {
		for _, issue := range slice {
			issues = append(issues, *issue)
		}
	}
	return issues, nil
}

// AssignIssue adds assignees.
func (f *SCMClient) AssignIssue(owner, repo string, number int, assignees []string) error {
	var m scmprovider.MissingUsers
	for _, a := range assignees {
		if a == "not-in-the-org" {
			m.Users = append(m.Users, a)
			continue
		}
		f.AssigneesAdded = append(f.AssigneesAdded, fmt.Sprintf("%s/%s#%d:%s", owner, repo, number, a))
	}
	if m.Users == nil {
		return nil
	}
	return m
}

// GetFile returns the bytes of the file.
func (f *SCMClient) GetFile(org, repo, file, commit string) ([]byte, error) {
	contents, ok := f.RemoteFiles[file]
	if !ok {
		return nil, fmt.Errorf("could not find file %s", file)
	}
	if commit == "" {
		if master, ok := contents["master"]; ok {
			return []byte(master), nil
		}

		return nil, fmt.Errorf("could not find file %s in master", file)
	}

	if content, ok := contents[commit]; ok {
		return []byte(content), nil
	}

	return nil, fmt.Errorf("could not find file %s with ref %s", file, commit)
}

// ListTeams return a list of fake teams that correspond to the fake team members returned by ListTeamMembers
func (f *SCMClient) ListTeams(org string) ([]*scm.Team, error) {
	return []*scm.Team{
		{
			ID:   0,
			Name: "Admins",
		},
		{
			ID:   42,
			Name: "Leads",
		},
	}, nil
}

// ListOrgMembers returns an empty list for now
func (f *SCMClient) ListOrgMembers(org string) ([]*scm.TeamMember, error) {
	return nil, nil
}

// ListTeamMembers return a fake team with a single "sig-lead" Github teammember
func (f *SCMClient) ListTeamMembers(teamID int, role string) ([]*scm.TeamMember, error) {
	if role != scmprovider.RoleAll {
		return nil, fmt.Errorf("unsupported role %v (only all supported)", role)
	}
	teams := map[int][]*scm.TeamMember{
		0:  {{Login: "default-sig-lead"}},
		42: {{Login: "sig-lead"}},
	}
	members, ok := teams[teamID]
	if !ok {
		return []*scm.TeamMember{}, nil
	}
	return members, nil
}

// IsCollaborator returns true if the user is a collaborator of the repo.
func (f *SCMClient) IsCollaborator(org, repo, login string) (bool, error) {
	normed := scmprovider.NormLogin(login)
	for _, collab := range f.Collaborators {
		if scmprovider.NormLogin(collab) == normed {
			return true, nil
		}
	}
	return false, nil
}

// ListCollaborators lists the collaborators.
func (f *SCMClient) ListCollaborators(org, repo string) ([]scm.User, error) {
	result := make([]scm.User, 0, len(f.Collaborators))
	for _, login := range f.Collaborators {
		result = append(result, scm.User{Login: login})
	}
	return result, nil
}

// ClearMilestone removes the milestone
func (f *SCMClient) ClearMilestone(org, repo string, issueNum int) error {
	f.Milestone = 0
	return nil
}

// SetMilestone sets the milestone.
func (f *SCMClient) SetMilestone(org, repo string, issueNum, milestoneNum int) error {
	if milestoneNum < 0 {
		return fmt.Errorf("Milestone Numbers Cannot Be Negative")
	}
	f.Milestone = milestoneNum
	return nil
}

// ListMilestones lists milestones.
func (f *SCMClient) ListMilestones(org, repo string) ([]scmprovider.Milestone, error) {
	milestones := []scmprovider.Milestone{}
	for k, v := range f.MilestoneMap {
		milestones = append(milestones, scmprovider.Milestone{Title: k, Number: v})
	}
	return milestones, nil
}

// ListPRCommits lists commits for a given PR.
func (f *SCMClient) ListPRCommits(org, repo string, prNumber int) ([]scm.Commit, error) {
	k := fmt.Sprintf("%s/%s#%d", org, repo, prNumber)
	return f.CommitMap[k], nil
}
