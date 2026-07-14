package fake

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/sets"
)

func newFake() *FakeRepoOwners {
	return &FakeRepoOwners{
		Dirs: map[string]DirOwners{
			"": {
				Approvers: sets.NewString("root-approver"),
				Reviewers: sets.NewString("root-reviewer"),
				Labels:    sets.NewString("root-label"),
			},
			"a": {
				Approvers:        sets.NewString("alice"),
				Reviewers:        sets.NewString("aria"),
				Labels:           sets.NewString("area/a"),
				MinimumReviewers: 2,
			},
			"a/b": {
				Approvers: sets.NewString("bob"),
				Reviewers: sets.NewString("bella"),
			},
			"c": {
				Approvers:      sets.NewString("carol"),
				NoParentOwners: true,
			},
		},
	}
}

func TestApproversWalksUp(t *testing.T) {
	f := newFake()
	assert.Equal(t, sets.NewString("root-approver", "alice", "bob"), f.Approvers("a/b/x.go"))
}

func TestLeafApproversStopsAtClosest(t *testing.T) {
	f := newFake()
	assert.Equal(t, sets.NewString("bob"), f.LeafApprovers("a/b/x.go"))
	assert.Equal(t, sets.NewString("alice"), f.LeafApprovers("a/x.go"))
}

func TestNoParentOwnersStopsWalk(t *testing.T) {
	f := newFake()
	assert.Equal(t, sets.NewString("carol"), f.Approvers("c/x.go"))
}

func TestFindApproverOwnersForFileSkipsRoot(t *testing.T) {
	f := newFake()
	assert.Equal(t, "a/b", f.FindApproverOwnersForFile("a/b/x.go"))
	assert.Equal(t, "a", f.FindApproverOwnersForFile("a/x.go"))
	// No non-root ancestor lists approvers for a top-level file, so we get "".
	assert.Equal(t, "", f.FindApproverOwnersForFile("x.go"))
}

func TestFindLabelsForFileWalksUp(t *testing.T) {
	f := newFake()
	assert.Equal(t, sets.NewString("root-label", "area/a"), f.FindLabelsForFile("a/x.go"))
}

func TestMinimumReviewersUsesClosestAncestor(t *testing.T) {
	f := newFake()
	assert.Equal(t, 2, f.MinimumReviewersForFile("a/b/x.go"))
	assert.Equal(t, 1, f.MinimumReviewersForFile("x.go"))
}

func TestIsNoParentOwners(t *testing.T) {
	f := newFake()
	assert.True(t, f.IsNoParentOwners("c"))
	assert.False(t, f.IsNoParentOwners("a"))
	assert.False(t, f.IsNoParentOwners("nonexistent"))
}

func TestClientReturnsOwners(t *testing.T) {
	owners := newFake()
	c := &Client{Owners: owners}
	got, err := c.LoadRepoOwners("org", "repo", "main")
	assert.NoError(t, err)
	assert.Same(t, owners, got)
}
