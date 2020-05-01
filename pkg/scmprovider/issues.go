package scmprovider

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/pkg/errors"
)

// RateLimits contains rate limit information
type RateLimits struct {
	Remaining int
	Limit     int
	Reset     int
}

func (r *RateLimits) populate(res *scm.Response) {
	if res == nil {
		return
	}
	h := res.Header

	r.Remaining = firstNumber(h["X-Ratelimit-Remaining"])
	r.Limit = firstNumber(h["X-Ratelimit-Limit"])
	r.Reset = firstNumber(h["X-Ratelimit-Reset"])
}

func firstNumber(values []string) int {
	if len(values) == 0 {
		return 0
	}
	answer, err := strconv.Atoi(values[0])
	if err != nil {
		answer = 0
	}
	return answer
}

// Query performs a GraphQL query on the git provider
func (c *Client) Query(ctx context.Context, q interface{}, vars map[string]interface{}) error {
	graphql := c.client.GraphQL
	if graphql == nil {
		log.Logger().Warnf("no GraphQL graphql supported for git provider %s", c.client.Driver.String())
		return nil
	}
	return graphql.Query(ctx, q, vars)
}

// Search query issues/PRs using a query string
func (c *Client) Search(opts scm.SearchOptions) ([]*scm.SearchIssue, *RateLimits, error) {
	ctx := context.Background()
	results, res, err := c.client.Issues.Search(ctx, opts)

	rates := &RateLimits{}
	rates.populate(res)
	return results, rates, err
}

// ListIssueEvents list issue events
func (c *Client) ListIssueEvents(org, repo string, number int) ([]*scm.ListedIssueEvent, error) {
	ctx := context.Background()
	fullName := c.repositoryName(org, repo)
	var allEvents []*scm.ListedIssueEvent
	var resp *scm.Response
	var events []*scm.ListedIssueEvent
	var err error
	firstRun := false
	opts := scm.ListOptions{
		Page: 1,
	}
	for !firstRun || (resp != nil && opts.Page <= resp.Page.Last) {
		events, resp, err = c.client.Issues.ListEvents(ctx, fullName, number, opts)
		if err != nil {
			return nil, err
		}
		firstRun = true
		allEvents = append(allEvents, events...)
		opts.Page++
	}
	return allEvents, nil
}

// AssignIssue assigns issue
func (c *Client) AssignIssue(owner, repo string, number int, logins []string) error {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	_, err := c.client.Issues.AssignIssue(ctx, fullName, number, logins)
	return err
}

// UnassignIssue unassigns issue
func (c *Client) UnassignIssue(owner, repo string, number int, logins []string) error {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	_, err := c.client.Issues.UnassignIssue(ctx, fullName, number, logins)
	return err
}

// AddLabel adds a label
func (c *Client) AddLabel(owner, repo string, number int, label string, pr bool) error {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	if pr {
		_, err := c.client.PullRequests.AddLabel(ctx, fullName, number, label)
		return err
	}
	_, err := c.client.Issues.AddLabel(ctx, fullName, number, label)
	return err
}

// RemoveLabel removes labesl
func (c *Client) RemoveLabel(owner, repo string, number int, label string, pr bool) error {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	if pr {
		_, err := c.client.PullRequests.DeleteLabel(ctx, fullName, number, label)
		return err
	}
	_, err := c.client.Issues.DeleteLabel(ctx, fullName, number, label)
	return err
}

// DeleteComment delete comments
func (c *Client) DeleteComment(org, repo string, number, ID int, pr bool) error {
	ctx := context.Background()
	fullName := c.repositoryName(org, repo)
	if pr {
		_, err := c.client.PullRequests.DeleteComment(ctx, fullName, number, ID)
		return err
	}
	_, err := c.client.Issues.DeleteComment(ctx, fullName, number, ID)
	return err
}

// DeleteStaleComments iterates over comments on an issue/PR, deleting those which the 'isStale'
// function identifies as stale. If 'comments' is nil, the comments will be fetched from GitHub.
func (c *Client) DeleteStaleComments(org, repo string, number int, comments []*scm.Comment, pr bool, isStale func(*scm.Comment) bool) error {
	var err error
	if comments == nil {
		if pr {
			comments, err = c.ListPullRequestComments(org, repo, number)
		} else {
			comments, err = c.ListIssueComments(org, repo, number)
		}
		if err != nil {
			return fmt.Errorf("failed to list comments while deleting stale comments. err: %v", err)
		}
	}
	for _, comment := range comments {
		if isStale(comment) {
			if err := c.DeleteComment(org, repo, number, comment.ID, pr); err != nil {
				return fmt.Errorf("failed to delete stale comment with ID '%d'", comment.ID)
			}
		}
	}
	return nil
}

// ListIssueComments list comments associated with an issue
func (c *Client) ListIssueComments(org, repo string, number int) ([]*scm.Comment, error) {
	ctx := context.Background()
	fullName := c.repositoryName(org, repo)
	var allComments []*scm.Comment
	var resp *scm.Response
	var comments []*scm.Comment
	var err error
	firstRun := false
	opts := scm.ListOptions{
		Page: 1,
	}
	for !firstRun || (resp != nil && opts.Page <= resp.Page.Last) {
		comments, resp, err = c.client.Issues.ListComments(ctx, fullName, number, opts)
		if err != nil {
			return nil, err
		}
		firstRun = true
		allComments = append(allComments, comments...)
		opts.Page++
	}
	return allComments, nil
}

// GetIssueLabels returns the issue labels
func (c *Client) GetIssueLabels(org, repo string, number int, pr bool) ([]*scm.Label, error) {
	ctx := context.Background()
	fullName := c.repositoryName(org, repo)
	var allLabels []*scm.Label
	var resp *scm.Response
	var labels []*scm.Label
	var err error
	firstRun := false
	opts := scm.ListOptions{
		Page: 1,
	}
	if pr {
		for !firstRun || (resp != nil && opts.Page <= resp.Page.Last) {
			labels, resp, err = c.client.PullRequests.ListLabels(ctx, fullName, number, opts)
			if err != nil {
				return nil, err
			}
			firstRun = true
			allLabels = append(allLabels, labels...)
			opts.Page++
		}
	}
	for !firstRun || (resp != nil && opts.Page <= resp.Page.Last) {
		labels, resp, err = c.client.Issues.ListLabels(ctx, fullName, number, opts)
		if err != nil {
			return nil, err
		}
		firstRun = true
		allLabels = append(allLabels, labels...)
		opts.Page++
	}
	return labels, err
}

// CreateComment create a comment
func (c *Client) CreateComment(owner, repo string, number int, pr bool, comment string) error {
	fullName := c.repositoryName(owner, repo)
	commentInput := scm.CommentInput{
		Body: comment,
	}
	ctx := context.Background()
	if pr {
		_, response, err := c.client.PullRequests.CreateComment(ctx, fullName, number, &commentInput)
		if err != nil {
			var b bytes.Buffer
			_, cperr := io.Copy(&b, response.Body)
			if cperr != nil {
				return errors.Wrapf(cperr, "response: %s", b.String())
			}
			return errors.Wrapf(err, "response: %s", b.String())
		}

	} else {
		_, response, err := c.client.Issues.CreateComment(ctx, fullName, number, &commentInput)
		if err != nil {
			var b bytes.Buffer
			_, cperr := io.Copy(&b, response.Body)
			if cperr != nil {
				return errors.Wrapf(cperr, "reponse: %s", b.String())
			}
			return errors.Wrapf(err, "response: %s", b.String())
		}
	}
	return nil
}

// EditComment edit a comment
func (c *Client) EditComment(owner, repo string, number int, id int, comment string, pr bool) error {
	fullName := c.repositoryName(owner, repo)
	commentInput := scm.CommentInput{
		Body: comment,
	}
	ctx := context.Background()
	if pr {
		_, response, err := c.client.PullRequests.EditComment(ctx, fullName, number, id, &commentInput)
		if err != nil {
			var b bytes.Buffer
			_, cperr := io.Copy(&b, response.Body)
			if cperr != nil {
				return errors.Wrapf(cperr, "response: %s", b.String())
			}
			return errors.Wrapf(err, "response: %s", b.String())
		}

	} else {
		_, response, err := c.client.Issues.EditComment(ctx, fullName, number, id, &commentInput)
		if err != nil {
			var b bytes.Buffer
			_, cperr := io.Copy(&b, response.Body)
			if cperr != nil {
				return errors.Wrapf(cperr, "reponse: %s", b.String())
			}
			return errors.Wrapf(err, "response: %s", b.String())
		}
	}
	return nil
}

// ReopenIssue reopen an issue
func (c *Client) ReopenIssue(owner, repo string, number int) error {
	panic("implement me")
}

// FindIssues find issues
func (c *Client) FindIssues(query, sort string, asc bool) ([]scm.Issue, error) {
	panic("implement me")
}

// CloseIssue close issue
func (c *Client) CloseIssue(owner, repo string, number int) error {
	panic("implement me")
}
