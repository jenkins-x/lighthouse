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

func (c *GitHubClient) GetSingleCommit(org, repo, SHA string) (scm.CommitTree, error) {
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

func (c *GitHubClient) RequestReview(org, repo string, number int, logins []string) error {
	panic("implement me")
}

func (c *GitHubClient) UnrequestReview(org, repo string, number int, logins []string) error {
	panic("implement me")
}

func (c *GitHubClient) GetPullRequest(owner, repo string, number int) (*scm.PullRequest, error) {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	pr, _, err := c.client.PullRequests.Find(ctx, fullName, number)
	return pr, err
}

func (c *GitHubClient) ListReviews(org, repo string, number int) ([]scm.Review, error) {
	panic("implement me")
}

func (c *GitHubClient) ListPullRequestComments(owner, repo string, number int) ([]*scm.Comment, error) {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	pr, _, err := c.client.PullRequests.ListComments(ctx, fullName, number, c.createListOptions())
	return pr, err
}

func (c *GitHubClient) BotName() (string, error) {
	panic("implement me")
}

func (c *GitHubClient) ListIssueEvents(org, repo string, num int) ([]ListedIssueEvent, error) {
	panic("implement me")
}

func (c *GitHubClient) AssignIssue(owner, repo string, number int, logins []string) error {
	panic("implement me")
}

func (c *GitHubClient) UnassignIssue(owner, repo string, number int, logins []string) error {
	panic("implement me")
}

func (c *GitHubClient) AddLabel(owner, repo string, number int, label string) error {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	_, err := c.client.Issues.AddLabel(ctx, fullName, number, label)
	return err
}

func (c *GitHubClient) RemoveLabel(owner, repo string, number int, label string) error {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	_, err := c.client.Issues.DeleteLabel(ctx, fullName, number, label)
	return err
}

func (c *GitHubClient) DeleteComment(org, repo string, number, ID int) error {
	ctx := context.Background()
	fullName := c.repositoryName(org, repo)
	_, err := c.client.Issues.DeleteComment(ctx, fullName, number, ID)
	return err
}

func (c *GitHubClient) ListIssueComments(org, repo string, number int) ([]*scm.Comment, error) {
	ctx := context.Background()
	fullName := c.repositoryName(org, repo)
	comments, _, err := c.client.Issues.ListComments(ctx, fullName, number, c.createListOptions())
	return comments, err
}

func (c *GitHubClient) GetIssueLabels(org, repo string, number int) ([]scm.Label, error) {
	ctx := context.Background()
	fullName := c.repositoryName(org, repo)
	issue, _, err := c.client.Issues.Find(ctx, fullName, number)
	if err != nil {
		return nil, err
	}
	// TODO we don't currently load all the isue label information...
	return toScmLabels(issue.Labels), nil
}

func toScmLabels(labels []string) []scm.Label {
	answer := []scm.Label{}
	for _, label := range labels {
		answer = append(answer, scm.Label{
			Name: label,
		})
	}
	return answer
}

func (c *GitHubClient) GetPullRequestChanges(org, repo string, number int) ([]*scm.Change, error) {
	ctx := context.Background()
	fullName := c.repositoryName(org, repo)
	changes, _, err := c.client.PullRequests.ListChanges(ctx, fullName, number, c.createListOptions())
	return changes, err
}

func (c *GitHubClient) CreateComment(owner, repo string, number int, comment string) error {
	fullName := c.repositoryName(owner, repo)
	commentInput := scm.CommentInput{
		Body: comment,
	}
	ctx := context.Background()
	_, response, err := c.client.Issues.CreateComment(ctx, fullName, number, &commentInput)
	if err != nil {
		var b bytes.Buffer
		io.Copy(&b, response.Body)
		return errors.Wrapf(err, "response: %s", b.String())
	}
	return nil
}

func (c *GitHubClient) repositoryName(owner string, repo string) string {
	return fmt.Sprintf("%s/%s", owner, repo)
}

func (c *GitHubClient) createListOptions() scm.ListOptions {
	return scm.ListOptions{}
}

// FileNotFound happens when github cannot find the file requested by GetFile().
type FileNotFound struct {
	org, repo, path, commit string
}

func (e *FileNotFound) Error() string {
	return fmt.Sprintf("%s/%s/%s @ %s not found", e.org, e.repo, e.path, e.commit)
}
