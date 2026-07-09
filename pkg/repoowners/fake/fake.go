// Package fake provides a shared test double for repoowners.RepoOwner
// that emulates the hierarchical OWNERS-file semantics of the real
// implementation. Tests supply the leaf OWNERS content per directory and
// the fake performs the same upward walk the real implementation does.
package fake

import (
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/jenkins-x/lighthouse/pkg/repoowners"
)

// DirOwners is the parsed content of a single OWNERS file at a directory.
// Fields left zero-valued behave as absent from the file.
type DirOwners struct {
	Approvers         sets.String
	Reviewers         sets.String
	RequiredReviewers sets.String
	Labels            sets.String
	// MinimumReviewers mirrors the OWNERS `minimum_reviewers` field.
	// A value of 0 means the field is unset for this directory.
	MinimumReviewers int
	// NoParentOwners mirrors the OWNERS `options.no_parent_owners` field:
	// when true, entry lookups stop walking upward past this directory.
	NoParentOwners bool
}

// FakeRepoOwners implements repoowners.RepoOwner with an in-memory,
// directory-keyed OWNERS tree. Given a file path it walks up parent
// directories the same way the real implementation does.
//
// Directory keys are canonical relative paths without leading or trailing
// slashes; the repo root is the empty string "".
//
// Files provides an optional per-file overlay: entries there behave like
// the closest OWNERS (real-impl analogue is a `.md` YAML header or a
// FullConfig regex filter that matches exactly one path). The upward
// directory walk still runs on top of a Files match.
type FakeRepoOwners struct {
	Dirs  map[string]DirOwners
	Files map[string]DirOwners
}

var _ repoowners.RepoOwner = (*FakeRepoOwners)(nil)

// Client implements repoowners.Interface. LoadRepoOwners always returns
// the same underlying FakeRepoOwners so tests can inspect state.
type Client struct {
	Owners *FakeRepoOwners
}

var _ repoowners.Interface = (*Client)(nil)

// LoadRepoAliases returns the aliases map. FakeRepoOwners does not model
// aliases separately from expanded owner sets; tests should supply
// already-expanded logins in DirOwners.
func (c *Client) LoadRepoAliases(org, repo, base string) (repoowners.RepoAliases, error) {
	return nil, nil
}

// LoadRepoOwners returns the shared FakeRepoOwners instance, allocating
// an empty one on demand so callers can rely on a non-nil return.
func (c *Client) LoadRepoOwners(org, repo, base string) (repoowners.RepoOwner, error) {
	if c.Owners == nil {
		c.Owners = &FakeRepoOwners{}
	}
	return c.Owners, nil
}

// canonicalize matches repoowners.canonicalize so upward walks land on the
// same directory keys the real implementation would produce.
func canonicalize(path string) string {
	if path == "." {
		return ""
	}
	return strings.TrimSuffix(path, "/")
}

// dirOf returns the canonical directory containing path.
func dirOf(path string) string {
	return canonicalize(filepath.Dir(path))
}

// entriesForFile walks upward from the file's directory, unioning the
// selected field from each DirOwners it visits. Stops at the repo root or
// at the first directory with NoParentOwners set. If leafOnly is true it
// returns as soon as any entries have been collected.
func (f *FakeRepoOwners) entriesForFile(path string, extract func(DirOwners) sets.String, leafOnly bool) sets.String {
	out := sets.NewString()
	if cfg, ok := f.Files[path]; ok {
		out.Insert(extract(cfg).UnsortedList()...)
		if leafOnly && out.Len() > 0 {
			return out
		}
		if cfg.NoParentOwners {
			return out
		}
	}
	d := path
	for {
		if cfg, ok := f.Dirs[d]; ok {
			out.Insert(extract(cfg).UnsortedList()...)
			if leafOnly && out.Len() > 0 {
				return out
			}
			if cfg.NoParentOwners {
				return out
			}
		}
		if d == "" {
			return out
		}
		d = dirOf(d)
	}
}

// findOwnersForFile returns the deepest path with a non-empty extracted
// field: first the file itself if it has a per-file overlay entry, then
// each ancestor directory. Matches the real implementation's behaviour
// where the repo-root OWNERS is not considered for this lookup.
func (f *FakeRepoOwners) findOwnersForFile(path string, extract func(DirOwners) sets.String) string {
	if cfg, ok := f.Files[path]; ok && extract(cfg).Len() > 0 {
		return path
	}
	d := path
	for d != "" {
		if cfg, ok := f.Dirs[d]; ok && extract(cfg).Len() > 0 {
			return d
		}
		d = dirOf(d)
	}
	return ""
}

// FindApproverOwnersForFile returns the deepest ancestor directory whose
// OWNERS file lists approvers, or "" if none is found (excluding root).
func (f *FakeRepoOwners) FindApproverOwnersForFile(path string) string {
	return f.findOwnersForFile(path, func(d DirOwners) sets.String { return d.Approvers })
}

// FindReviewersOwnersForFile returns the deepest ancestor directory whose
// OWNERS file lists reviewers, or "" if none is found (excluding root).
func (f *FakeRepoOwners) FindReviewersOwnersForFile(path string) string {
	return f.findOwnersForFile(path, func(d DirOwners) sets.String { return d.Reviewers })
}

// FindLabelsForFile returns the union of labels applied to path by
// OWNERS files in the file's directory and its ancestors.
func (f *FakeRepoOwners) FindLabelsForFile(path string) sets.String {
	return f.entriesForFile(path, func(d DirOwners) sets.String { return d.Labels }, false)
}

// IsNoParentOwners reports whether the OWNERS file at dir has the
// no_parent_owners option set.
func (f *FakeRepoOwners) IsNoParentOwners(dir string) bool {
	return f.Dirs[canonicalize(dir)].NoParentOwners
}

// LeafApprovers returns approvers from the closest OWNERS ancestor only.
func (f *FakeRepoOwners) LeafApprovers(path string) sets.String {
	return f.entriesForFile(path, func(d DirOwners) sets.String { return d.Approvers }, true)
}

// Approvers returns approvers from every OWNERS ancestor of path.
func (f *FakeRepoOwners) Approvers(path string) sets.String {
	return f.entriesForFile(path, func(d DirOwners) sets.String { return d.Approvers }, false)
}

// LeafReviewers returns reviewers from the closest OWNERS ancestor only.
func (f *FakeRepoOwners) LeafReviewers(path string) sets.String {
	return f.entriesForFile(path, func(d DirOwners) sets.String { return d.Reviewers }, true)
}

// Reviewers returns reviewers from every OWNERS ancestor of path.
func (f *FakeRepoOwners) Reviewers(path string) sets.String {
	return f.entriesForFile(path, func(d DirOwners) sets.String { return d.Reviewers }, false)
}

// RequiredReviewers returns required reviewers from every OWNERS ancestor of path.
func (f *FakeRepoOwners) RequiredReviewers(path string) sets.String {
	return f.entriesForFile(path, func(d DirOwners) sets.String { return d.RequiredReviewers }, false)
}

// MinimumReviewersForFile walks up from path returning the closest
// MinimumReviewers value; returns 1 (the real implementation's default)
// if no ancestor sets one. The walk starts at path itself so callers
// that pass a dir path (rather than a file path) get the entry at that
// directory rather than at its parent.
func (f *FakeRepoOwners) MinimumReviewersForFile(path string) int {
	d := path
	for {
		if cfg, ok := f.Dirs[d]; ok {
			if cfg.MinimumReviewers > 0 {
				return cfg.MinimumReviewers
			}
			if cfg.NoParentOwners {
				return 1
			}
		}
		if d == "" {
			return 1
		}
		d = dirOf(d)
	}
}
