package scmprovider

import (
	"context"
	"fmt"
	"os"

	"github.com/jenkins-x/go-scm/scm"
	"k8s.io/apimachinery/pkg/util/sets"
)

// ToClient converts the scm client to an API that the prow plugins expect
func ToClient(client *scm.Client, botName string) *Client {
	return &Client{client: client, botName: botName}
}

// SCMClient is an interface providing all functions on the Client struct.
type SCMClient interface {
	// Functions implemented in client.go
	BotName() (string, error)
	SetBotName(string)
	SupportsGraphQL() bool
	ProviderType() string
	SupportsPRLabels() bool

	// Functions implemented in content.go
	GetFile(string, string, string, string) ([]byte, error)

	// Functions implemented in git.go
	GetRef(string, string, string) (string, error)
	DeleteRef(string, string, string) error
	GetSingleCommit(string, string, string) (*scm.Commit, error)

	// Functions implemented in issues.go
	Query(context.Context, interface{}, map[string]interface{}) error
	Search(scm.SearchOptions) ([]*scm.SearchIssue, *RateLimits, error)
	ListIssueEvents(string, string, int) ([]*scm.ListedIssueEvent, error)
	AssignIssue(string, string, int, []string) error
	UnassignIssue(string, string, int, []string) error
	AddLabel(string, string, int, string, bool) error
	RemoveLabel(string, string, int, string, bool) error
	DeleteComment(string, string, int, int, bool) error
	DeleteStaleComments(string, string, int, []*scm.Comment, bool, func(*scm.Comment) bool) error
	ListIssueComments(string, string, int) ([]*scm.Comment, error)
	GetIssueLabels(string, string, int, bool) ([]*scm.Label, error)
	CreateComment(string, string, int, bool, string) error
	ReopenIssue(string, string, int) error
	FindIssues(string, string, bool) ([]scm.Issue, error)
	CloseIssue(string, string, int) error
	EditComment(owner, repo string, number int, id int, comment string, pr bool) error

	// Functions implemented in organizations.go
	ListTeams(string) ([]*scm.Team, error)
	ListTeamMembers(int, string) ([]*scm.TeamMember, error)
	ListOrgMembers(string) ([]*scm.TeamMember, error)
	IsOrgAdmin(string, string) (bool, error)

	// Functions implemented in pull_requests.go
	GetPullRequest(string, string, int) (*scm.PullRequest, error)
	ListPullRequestComments(string, string, int) ([]*scm.Comment, error)
	GetPullRequestChanges(string, string, int) ([]*scm.Change, error)
	Merge(string, string, int, MergeDetails) error
	ReopenPR(string, string, int) error
	ClosePR(string, string, int) error
	ListAllPullRequestsForFullNameRepo(string, scm.PullRequestListOptions) ([]*scm.PullRequest, error)

	// Functions implemented in repositories.go
	GetRepoLabels(string, string) ([]*scm.Label, error)
	IsCollaborator(string, string, string) (bool, error)
	ListCollaborators(string, string) ([]scm.User, error)
	CreateStatus(string, string, string, *scm.StatusInput) (*scm.Status, error)
	CreateGraphQLStatus(string, string, string, *Status) (*scm.Status, error)
	ListStatuses(string, string, string) ([]*scm.Status, error)
	GetCombinedStatus(string, string, string) (*scm.CombinedStatus, error)
	HasPermission(string, string, string, ...string) (bool, error)
	GetUserPermission(string, string, string) (string, error)
	IsMember(string, string) (bool, error)
	GetRepositoryByFullName(string) (*scm.Repository, error)

	// Functions implemented in reviews.go
	ListReviews(string, string, int) ([]*scm.Review, error)
	RequestReview(string, string, int, []string) error
	UnrequestReview(string, string, int, []string) error

	// Functions not yet implemented
	ClearMilestone(string, string, int) error
	SetMilestone(string, string, int, int) error
	ListMilestones(string, string) ([]Milestone, error)
}

// Client represents an interface that prow plugins expect on top of go-scm
type Client struct {
	client  *scm.Client
	botName string
}

// ClearMilestone clears milestone
func (c *Client) ClearMilestone(org, repo string, num int) error {
	return scm.ErrNotSupported
}

// SetMilestone sets milestone
func (c *Client) SetMilestone(org, repo string, issueNum, milestoneNum int) error {
	return scm.ErrNotSupported
}

// ListMilestones list milestones
func (c *Client) ListMilestones(org, repo string) ([]Milestone, error) {
	return nil, scm.ErrNotSupported
}

// BotName returns the bot name
func (c *Client) BotName() (string, error) {
	botName := c.botName
	if botName == "" {
		botName = os.Getenv("GIT_USER")
		if botName == "" {
			botName = "jenkins-x-bot"
		}
		c.botName = botName
	}
	return botName, nil
}

// SetBotName sets the bot name
func (c *Client) SetBotName(botName string) {
	c.botName = botName
}

// SupportsPRLabels returns true if the underlying provider supports PR labels
func (c *Client) SupportsPRLabels() bool {
	return !NoLabelProviders().Has(c.ProviderType())
}

// SupportsGraphQL returns true if the underlying provider supports our GraphQL queries
func (c *Client) SupportsGraphQL() bool {
	return c.client.GraphQL != nil
}

// ProviderType returns the type of the underlying SCM provider
func (c *Client) ProviderType() string {
	return c.client.Driver.String()
}

func (c *Client) repositoryName(owner string, repo string) string {
	return fmt.Sprintf("%s/%s", owner, repo)
}

func (c *Client) createListOptions() scm.ListOptions {
	return scm.ListOptions{}
}

// FileNotFound happens when github cannot find the file requested by GetFile().
type FileNotFound struct {
	org, repo, path, commit string
}

// Error formats a file not found error
func (e *FileNotFound) Error() string {
	return fmt.Sprintf("%s/%s/%s @ %s not found", e.org, e.repo, e.path, e.commit)
}

// NoLabelProviders returns a set of provider names that don't support labels.
func NoLabelProviders() sets.String {
	// "coding" is a placeholder provider name from go-scm that we'll use for testing the comment support for label logic.
	return sets.NewString("stash", "coding")
}
