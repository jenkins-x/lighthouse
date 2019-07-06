package github

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jenkins-x/go-scm/scm"
)

const (
	// RoleAll lists both members and admins
	RoleAll = "all"
	// RoleAdmin specifies the user is an org admin, or lists only admins
	RoleAdmin = "admin"
	// RoleMaintainer specifies the user is a team maintainer, or lists only maintainers
	RoleMaintainer = "maintainer"
	// RoleMember specifies the user is a regular user, or only lists regular users
	RoleMember = "member"
	// StatePending specifies the user has an invitation to the org/team.
	StatePending = "pending"
	// StateActive specifies the user's membership is active.
	StateActive = "active"
)

const (
	// EventGUID is sent by Github in a header of every webhook request.
	// Used as a log field across prow.
	EventGUID = "event-GUID"
	// PrLogField is the number of a PR.
	// Used as a log field across prow.
	PrLogField = "pr"
	// OrgLogField is the organization of a PR.
	// Used as a log field across prow.
	OrgLogField = "org"
	// RepoLogField is the repository of a PR.
	// Used as a log field across prow.
	RepoLogField = "repo"

	// SearchTimeFormat is a time.Time format string for ISO8601 which is the
	// format that GitHub requires for times specified as part of a search query.
	SearchTimeFormat = "2006-01-02T15:04:05Z"
)

// GenericCommentEventAction coerces multiple actions into its generic equivalent.
type GenericCommentEventAction string

// Comments indicate values that are coerced to the specified value.
const (
	// GenericCommentActionCreated means something was created/opened/submitted
	GenericCommentActionCreated GenericCommentEventAction = "created" // "opened", "submitted"
	// GenericCommentActionEdited means something was edited.
	GenericCommentActionEdited = "edited"
	// GenericCommentActionDeleted means something was deleted/dismissed.
	GenericCommentActionDeleted = "deleted" // "dismissed"
)

type PullRequestMergeType string

// Possible types of merges for the GitHub merge API
const (
	MergeMerge  PullRequestMergeType = "merge"
	MergeRebase PullRequestMergeType = "rebase"
	MergeSquash PullRequestMergeType = "squash"
)

// NormLogin normalizes GitHub login strings
var NormLogin = strings.ToLower

/*// HasLabel checks if label is in the label set "issueLabels".
func HasLabel(label string, issueLabels []scm.Label) bool {
	for _, l := range issueLabels {
		if strings.ToLower(l.Name) == strings.ToLower(label) {
			return true
		}
	}
	return false
}
*/
// ImageTooBig checks if image is bigger than github limits
func ImageTooBig(url string) (bool, error) {
	// limit is 10MB
	limit := 10000000
	// try to get the image size from Content-Length header
	resp, err := http.Head(url)
	if err != nil {
		return true, fmt.Errorf("HEAD error: %v", err)
	}
	if sc := resp.StatusCode; sc != http.StatusOK {
		return true, fmt.Errorf("failing %d response", sc)
	}
	size, _ := strconv.Atoi(resp.Header.Get("Content-Length"))
	if size > limit {
		return true, nil
	}
	return false, nil
}

// PullRequestChange contains information about what a PR changed.
type PullRequestChange struct {
	Sha              string `json:"sha"`
	Filename         string `json:"filename"`
	Status           string `json:"status"`
	Additions        int    `json:"additions"`
	Deletions        int    `json:"deletions"`
	Changes          int    `json:"changes"`
	Patch            string `json:"patch"`
	BlobURL          string `json:"blob_url"`
	PreviousFilename string `json:"previous_filename"`
}

// ReviewComment describes a Pull Request review.
type ReviewComment struct {
	ID        int       `json:"id"`
	ReviewID  int       `json:"pull_request_review_id"`
	User      scm.User  `json:"user"`
	Body      string    `json:"body"`
	Path      string    `json:"path"`
	HTMLURL   string    `json:"html_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// Position will be nil if the code has changed such that the comment is no
	// longer relevant.
	Position *int `json:"position"`
}

// CombinedStatus is the latest statuses for a ref.
type CombinedStatus struct {
	SHA      string       `json:"sha"`
	Statuses []scm.Status `json:"statuses"`
}

// ListedIssueEvent represents an issue event from the events API (not from a webhook payload).
// https://developer.github.com/v3/issues/events/
type ListedIssueEvent struct {
	Event   string    `json:"event"` // This is the same as IssueEvent.Action.
	Actor   scm.User  `json:"actor"`
	Label   scm.Label `json:"label"`
	Created time.Time `json:"created_at"`
}

// IssueEventAction enumerates the triggers for this
// webhook payload type. See also:
// https://developer.github.com/v3/activity/events/types/#issuesevent
type IssueEventAction string

const (
	// IssueActionAssigned means assignees were added.
	IssueActionAssigned IssueEventAction = "assigned"
	// IssueActionUnassigned means assignees were added.
	IssueActionUnassigned = "unassigned"
	// IssueActionLabeled means labels were added.
	IssueActionLabeled = "labeled"
	// IssueActionUnlabeled means labels were removed.
	IssueActionUnlabeled = "unlabeled"
	// IssueActionOpened means issue was opened/created.
	IssueActionOpened = "opened"
	// IssueActionEdited means issue body was edited.
	IssueActionEdited = "edited"
	// IssueActionMilestoned means the milestone was added/changed.
	IssueActionMilestoned = "milestoned"
	// IssueActionDemilestoned means a milestone was removed.
	IssueActionDemilestoned = "demilestoned"
	// IssueActionClosed means issue was closed.
	IssueActionClosed = "closed"
	// IssueActionReopened means issue was reopened.
	IssueActionReopened = "reopened"
)

// SingleCommit is the commit part received when requesting a single commit
// https://developer.github.com/v3/repos/commits/#get-a-single-commit
type SingleCommit struct {
	Commit struct {
		Tree struct {
			SHA string `json:"sha"`
		} `json:"tree"`
	} `json:"commit"`
}

// Team is a github organizational team
type Team struct {
	ID           int    `json:"id,omitempty"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	Privacy      string `json:"privacy,omitempty"`
	Parent       *Team  `json:"parent,omitempty"`         // Only present in responses
	ParentTeamID *int   `json:"parent_team_id,omitempty"` // Only valid in creates/edits
}

// TeamMember is a member of an organizational team
type TeamMember struct {
	Login string `json:"login"`
}

// GenericCommentEvent is a fake event type that is instantiated for any github event that contains
// comment like content.
// The specific events that are also handled as GenericCommentEvents are:
// - issue_comment events
// - pull_request_review events
// - pull_request_review_comment events
// - pull_request events with action in ["opened", "edited"]
// - issue events with action in ["opened", "edited"]
//
// Issue and PR "closed" events are not coerced to the "deleted" Action and do not trigger
// a GenericCommentEvent because these events don't actually remove the comment content from GH.
type GenericCommentEvent struct {
	IsPR        bool
	Action      scm.Action
	Body        string
	Link        string
	Number      int
	Repo        scm.Repository
	Author      scm.User
	IssueAuthor scm.User
	Assignees   []scm.User
	IssueState  string
	IssueBody   string
	IssueLink   string
	GUID        string
}

// RepositoryCommit represents a commit in a repo.
// Note that it's wrapping a GitCommit, so author/committer information is in two places,
// but contain different details about them: in RepositoryCommit "github details", in GitCommit - "git details".
type RepositoryCommit struct {
	SHA         string       `json:"sha"`
	Commit      GitCommit    `json:"commit"`
	Author      scm.User     `json:"author"`
	Committer   scm.User     `json:"committer"`
	Parents     []scm.Commit `json:"parents,omitempty"`
	HTMLURL     string       `json:"html_url"`
	URL         string       `json:"url"`
	CommentsURL string       `json:"comments_url"`
}

// GitCommit represents a GitHub commit.
type GitCommit struct {
	SHA     string `json:"sha,omitempty"`
	Message string `json:"message,omitempty"`
}

// ReviewAction is the action that a review can be made with.
type ReviewAction string

// Possible review actions. Leave Action blank for a pending review.
const (
	Approve        ReviewAction = "APPROVE"
	RequestChanges              = "REQUEST_CHANGES"
	Comment                     = "COMMENT"
)

// DraftReview is what we give GitHub when we want to make a PR Review. This is
// different than what we receive when we ask for a Review.
type DraftReview struct {
	// If unspecified, defaults to the most recent commit in the PR.
	CommitSHA string `json:"commit_id,omitempty"`
	Body      string `json:"body"`
	// If unspecified, defaults to PENDING.
	Action   ReviewAction         `json:"event,omitempty"`
	Comments []DraftReviewComment `json:"comments,omitempty"`
}

// DraftReviewComment is a comment in a draft review.
type DraftReviewComment struct {
	Path string `json:"path"`
	// Position in the patch, not the line number in the file.
	Position int    `json:"position"`
	Body     string `json:"body"`
}

// MissingUsers is an error specifying the users that could not be unassigned.
type MissingUsers struct {
	Users  []string
	action string
}

func (m MissingUsers) Error() string {
	return fmt.Sprintf("could not %s the following user(s): %s.", m.action, strings.Join(m.Users, ", "))
}

// Milestone is a milestone defined on a github repository
type Milestone struct {
	Title  string `json:"title"`
	Number int    `json:"number"`
}

// HasLabel checks if label is in the label set "issueLabels".
func HasLabel(label string, issueLabels []scm.Label) bool {
	for _, l := range issueLabels {
		if strings.ToLower(l.Name) == strings.ToLower(label) {
			return true
		}
	}
	return false
}

// Possible contents for reactions.
const (
	ReactionThumbsUp                  = "+1"
	ReactionThumbsDown                = "-1"
	ReactionLaugh                     = "laugh"
	ReactionConfused                  = "confused"
	ReactionHeart                     = "heart"
	ReactionHooray                    = "hooray"
	stateCannotBeChangedMessagePrefix = "state cannot be changed."
)

// These are possible State entries for a Status.
const (
	StatusPending = "pending"
	StatusSuccess = "success"
	StatusError   = "error"
	StatusFailure = "failure"
)

// Status is used to set a commit status line.
type Status struct {
	State       string `json:"state"`
	TargetURL   string `json:"target_url,omitempty"`
	Description string `json:"description,omitempty"`
	Context     string `json:"context,omitempty"`
}
