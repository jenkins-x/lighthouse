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

// Package reporter contains helpers for writing comments in scm providers.
package reporter

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
)

const (
	commentTag = "!-- test report --"
)

// SCMProviderClient provides a client interface to report job status updates
// through GitHub comments.
type SCMProviderClient interface {
	BotName() (string, error)
	ListPullRequestComments(string, string, int) ([]*scm.Comment, error)
	CreateComment(string, string, int, bool, string) error
	DeleteComment(string, string, int, int, bool) error
	EditComment(string, string, int, int, string, bool) error
}

// ShouldReport determines whether this LighthouseJob is of a type to be reporting back.
func ShouldReport(lhj *v1alpha1.LighthouseJob, validTypes []v1alpha1.PipelineKind) bool {
	valid := false
	for _, t := range validTypes {
		if lhj.Spec.Type == t {
			valid = true
		}
	}

	if !valid {
		return false
	}

	return true
}

// Report is creating/updating/removing report comments in the SCM provider based on the state of
// the provided LighthouseJob.
func Report(spc SCMProviderClient, reportTemplate *template.Template, lhj *v1alpha1.LighthouseJob, validTypes []v1alpha1.PipelineKind) error {
	if spc == nil {
		return fmt.Errorf("trying to report lhj %s, but found empty SCM provider client", lhj.ObjectMeta.Name)
	}

	if !ShouldReport(lhj, validTypes) {
		return nil
	}

	refs := lhj.Spec.Refs
	// we are not reporting for batch jobs, we can consider support that in the future
	if len(refs.Pulls) > 1 {
		return nil
	}

	if lhj.Status.CompletionTime == nil {
		return nil
	}

	if len(refs.Pulls) == 0 {
		return nil
	}

	prcs, err := spc.ListPullRequestComments(refs.Org, refs.Repo, refs.Pulls[0].Number)
	if err != nil {
		return fmt.Errorf("error listing comments: %v", err)
	}
	botName, err := spc.BotName()
	if err != nil {
		return fmt.Errorf("error getting bot name: %v", err)
	}
	deletes, entries, updateID := parsePRComments(lhj, botName, prcs)
	for _, delete := range deletes {
		if err := spc.DeleteComment(refs.Org, refs.Repo, refs.Pulls[0].Number, delete, true); err != nil {
			return fmt.Errorf("error deleting comment: %v", err)
		}
	}
	if len(entries) > 0 {
		comment, err := createComment(reportTemplate, lhj, entries)
		if err != nil {
			return fmt.Errorf("generating comment: %v", err)
		}
		if updateID == 0 {
			if err := spc.CreateComment(refs.Org, refs.Repo, refs.Pulls[0].Number, true, comment); err != nil {
				return fmt.Errorf("error creating comment: %v", err)
			}
		} else {
			if err := spc.EditComment(refs.Org, refs.Repo, refs.Pulls[0].Number, updateID, comment, true); err != nil {
				return fmt.Errorf("error updating comment: %v", err)
			}
		}
	}
	return nil
}

// parsePRComments returns a list of comments to delete, a list of table
// entries, and the ID of the comment to update. If there are no table entries
// then don't make a new comment. Otherwise, if the comment to update is 0,
// create a new comment.
func parsePRComments(lhj *v1alpha1.LighthouseJob, botName string, ics []*scm.Comment) ([]int, []string, int) {
	var toDelete []int
	var previousComments []int
	var latestComment int
	var entries []string
	// First accumulate result entries and comment IDs
	for _, ic := range ics {
		if ic.Author.Login != botName {
			continue
		}
		// Old report comments started with the context. Delete them.
		// TODO(spxtr): Delete this check a few weeks after this merges.
		if strings.HasPrefix(ic.Body, lhj.Spec.Context) {
			toDelete = append(toDelete, ic.ID)
		}
		if !strings.Contains(ic.Body, commentTag) {
			continue
		}
		if latestComment != 0 {
			previousComments = append(previousComments, latestComment)
		}
		latestComment = ic.ID
		var tracking bool
		for _, line := range strings.Split(ic.Body, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "---") {
				tracking = true
			} else if len(line) == 0 {
				tracking = false
			} else if tracking {
				entries = append(entries, line)
			}
		}
	}
	var newEntries []string
	// Next decide which entries to keep.
	for i := range entries {
		keep := true
		f1 := strings.Split(entries[i], " | ")
		for j := range entries {
			if i == j {
				continue
			}
			f2 := strings.Split(entries[j], " | ")
			// Use the newer results if there are multiple.
			if j > i && f2[0] == f1[0] {
				keep = false
			}
		}
		// Use the current result if there is an old one.
		if lhj.Spec.Context == f1[0] {
			keep = false
		}
		if keep {
			newEntries = append(newEntries, entries[i])
		}
	}
	var createNewComment bool
	if lhj.Status.State == v1alpha1.FailureState {
		newEntries = append(newEntries, createEntry(lhj))
		createNewComment = true
	}
	toDelete = append(toDelete, previousComments...)
	if (createNewComment || len(newEntries) == 0) && latestComment != 0 {
		toDelete = append(toDelete, latestComment)
		latestComment = 0
	}
	return toDelete, newEntries, latestComment
}

func createEntry(lhj *v1alpha1.LighthouseJob) string {
	return strings.Join([]string{
		lhj.Spec.Context,
		lhj.Spec.Refs.Pulls[0].SHA,
		fmt.Sprintf("[link](%s)", lhj.Status.ReportURL),
		fmt.Sprintf("`%s`", lhj.Spec.RerunCommand),
	}, " | ")
}

// createComment take a LighthouseJob and a list of entries generated with
// createEntry and returns a nicely formatted comment. It may fail if template
// execution fails.
func createComment(reportTemplate *template.Template, lhj *v1alpha1.LighthouseJob, entries []string) (string, error) {
	plural := ""
	if len(entries) > 1 {
		plural = "s"
	}
	var b bytes.Buffer
	if reportTemplate != nil {
		if err := reportTemplate.Execute(&b, &lhj); err != nil {
			return "", err
		}
	}
	lines := []string{
		fmt.Sprintf("@%s: The following test%s **failed**, say `/retest` to rerun them all:", lhj.Spec.Refs.Pulls[0].Author, plural),
		"",
		"Test name | Commit | Details | Rerun command",
		"--- | --- | --- | ---",
	}
	lines = append(lines, entries...)
	if reportTemplate != nil {
		lines = append(lines, "", b.String())
	}
	lines = append(lines, []string{
		"",
		"<details>",
		"",
		plugins.AboutThisBot,
		"</details>",
		"<" + commentTag + ">",
	}...)
	return strings.Join(lines, "\n"), nil
}
