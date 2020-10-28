package scmprovider

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

// NormLogin normalizes GitHub login strings
var NormLogin = strings.ToLower

/*// HasLabel checks if label is in the label set "issueLabels".
func HasLabel(label string, issueLabels []*scm.Label) bool {
	for _, l := range issueLabels {
		if strings.ToLower(l.Name) == strings.ToLower(label) {
			return true
		}
	}
	return false
}
*/

var (
	// FoundingYear is the year GitHub was founded. This is just used so that
	// we can lower bound dates related to PRs and issues.
	FoundingYear, _ = time.Parse(SearchTimeFormat, "2007-01-01T00:00:00Z")
)

// ImageTooBig checks if image is bigger than github limits
func ImageTooBig(url string) (bool, error) {
	// limit is 10MB
	limit := 10000000
	// try to get the image size from Content-Length header
	resp, err := http.Head(url) // #nosec
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
	HeadSha     string
}

// ReviewAction is the action that a review can be made with.
type ReviewAction string

// Possible review actions. Leave Action blank for a pending review.
const (
	Approve        ReviewAction = "APPROVE"
	RequestChanges ReviewAction = "REQUEST_CHANGES"
	Comment        ReviewAction = "COMMENT"
)

// These are possible State entries for a Status.
const (
	StatusPending = "pending"
	StatusSuccess = "success"
	StatusError   = "error"
	StatusFailure = "failure"
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

// HasLabel checks if label is in the label set "issueLabels".
func HasLabel(label string, issueLabels []*scm.Label) bool {
	for _, l := range issueLabels {
		if strings.EqualFold(l.Name, label) {
			return true
		}
	}
	return false
}

// PushHookBranch returns the name of the branch to which the user pushed.
func PushHookBranch(pe *scm.PushHook) string {
	ref := strings.TrimPrefix(pe.Ref, "refs/heads/") // if Ref is a branch
	ref = strings.TrimPrefix(ref, "refs/tags/")      // if Ref is a tag
	return ref
}
