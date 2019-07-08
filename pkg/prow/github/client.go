package github

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/pkg/errors"
)

// ToGitHubClient converts the scm client to an API that the prow plgins expect
func ToGitHubClient(client *scm.Client) *GitHubClient {
	return &GitHubClient{client}
}

// GitHubClient represents an interface that prow plugins expect on top of go-scm
type GitHubClient struct {
	client *scm.Client
}

func (c *GitHubClient) ClearMilestone(org, repo string, num int) error {
	panic("implement me")
}

func (c *GitHubClient) SetMilestone(org, repo string, issueNum, milestoneNum int) error {
	panic("implement me")
}

func (c *GitHubClient) ListMilestones(org, repo string) ([]Milestone, error) {
	panic("implement me")
}

func (c *GitHubClient) GetFile(org, repo, filepath, commit string) ([]byte, error) {
	panic("implement me")
}

func (c *GitHubClient) FindIssues(query, sort string, asc bool) ([]scm.Issue, error) {
	panic("implement me")
}

func (c *GitHubClient) CloseIssue(owner, repo string, number int) error {
	panic("implement me")
}

func (c *GitHubClient) ClosePR(owner, repo string, number int) error {
	panic("implement me")
}

func (c *GitHubClient) ReopenIssue(owner, repo string, number int) error {
	panic("implement me")
}

func (c *GitHubClient) ReopenPR(owner, repo string, number int) error {
	panic("implement me")
}

func (c *GitHubClient) GetRepoLabels(owner, repo string) ([]scm.Label, error) {
	panic("implement me")
}

func (c *GitHubClient) IsCollaborator(owner, repo, login string) (bool, error) {
	panic("implement me")
}

func (c *GitHubClient) GetSingleCommit(org, repo, SHA string) (SingleCommit, error) {
	panic("implement me")
}

func (c *GitHubClient) IsMember(org, user string) (bool, error) {
	panic("implement me")
}

func (c *GitHubClient) ListTeams(org string) ([]Team, error) {
	panic("implement me")
}

func (c *GitHubClient) ListTeamMembers(id int, role string) ([]TeamMember, error) {
	panic("implement me")
}

func (c *GitHubClient) DeleteRef(owner, repo, ref string) error {
	panic("implement me")
}

func (c *GitHubClient) Query(context.Context, interface{}, map[string]interface{}) error {
	panic("implement me")
}

func (c *GitHubClient) AssignIssue(owner, repo string, number int, logins []string) error {
	panic("implement me")
}

func (c *GitHubClient) UnassignIssue(owner, repo string, number int, logins []string) error {
	panic("implement me")
}

func (c *GitHubClient) RequestReview(org, repo string, number int, logins []string) error {
	panic("implement me")
}

func (c *GitHubClient) UnrequestReview(org, repo string, number int, logins []string) error {
	panic("implement me")
}

func (c *GitHubClient) GetPullRequest(org, repo string, number int) (*scm.PullRequest, error) {
	panic("implement me")
}

func (c *GitHubClient) ListIssueComments(org, repo string, number int) ([]scm.Comment, error) {
	panic("implement me")
}

func (c *GitHubClient) ListReviews(org, repo string, number int) ([]scm.Review, error) {
	panic("implement me")
}

func (c *GitHubClient) ListPullRequestComments(org, repo string, number int) ([]scm.Review, error) {
	panic("implement me")
}

func (c *GitHubClient) DeleteComment(org, repo string, ID int) error {
	panic("implement me")
}

func (c *GitHubClient) BotName() (string, error) {
	panic("implement me")
}

func (c *GitHubClient) ListIssueEvents(org, repo string, num int) ([]ListedIssueEvent, error) {
	panic("implement me")
}

func (c *GitHubClient) AddLabel(owner, repo string, number int, label string) error {
	panic("implement me")
}

func (c *GitHubClient) RemoveLabel(owner, repo string, number int, label string) error {
	panic("implement me")
}

func (c *GitHubClient) GetIssueLabels(org, repo string, number int) ([]scm.Label, error) {
	panic("implement me")
}

func (c *GitHubClient) GetPullRequestChanges(org, repo string, number int) ([]scm.Change, error) {
	panic("implement me")
}

func (c *GitHubClient) CreateComment(owner, repo string, number int, comment string) error {
	fullName := fmt.Sprintf("%s/%s", owner, repo)
	commentInput := scm.CommentInput{
		Body: comment,
	}
	_, response, err := c.client.Issues.CreateComment(context.Background(), fullName, number, &commentInput)
	if err != nil {
		var b bytes.Buffer
		io.Copy(&b, response.Body)
		return errors.Wrapf(err, "response: %s", b.String())
	}
	return nil
}

// FileNotFound happens when github cannot find the file requested by GetFile().
type FileNotFound struct {
	org, repo, path, commit string
}

func (e *FileNotFound) Error() string {
	return fmt.Sprintf("%s/%s/%s @ %s not found", e.org, e.repo, e.path, e.commit)
}
