package scmprovider

import (
	"fmt"
	"strings"

	"github.com/jenkins-x/go-scm/scm"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	labelTag = "!-- label report --"
)

// GetLabelsFromComment extracts the applied "labels" from a pull request from a comment
func GetLabelsFromComment(spc SCMClient, org, repo string, number int) ([]*scm.Label, error) {
	comment, err := GetLabelReportComment(spc, org, repo, number)
	if err != nil {
		return nil, err
	}

	var labels []*scm.Label

	for _, l := range parseLabelReportComment(comment).List() {
		labels = append(labels, &scm.Label{
			Name: l,
		})
	}

	return labels, nil
}

// DeleteLabelFromComment removes a label from the tracking comment
func DeleteLabelFromComment(spc SCMClient, org, repo string, number int, label string) error {
	comment, err := GetLabelReportComment(spc, org, repo, number)
	if err != nil {
		return err
	}
	existingLabels := parseLabelReportComment(comment)

	existingLabels.Delete(label)

	if existingLabels.Len() == 0 {
		if comment != nil {
			return spc.DeleteComment(org, repo, number, comment.ID, true)
		}
		// If there isn't a comment and there are no labels, we've never really labeled in the first place so hey.
		return nil
	}
	newCommentBody := CreateLabelComment(existingLabels.List())

	if comment != nil && comment.Body != newCommentBody {
		return spc.EditComment(org, repo, number, comment.ID, newCommentBody, true)
	}
	if comment == nil {
		return spc.CreateComment(org, repo, number, true, newCommentBody)
	}
	return nil
}

// AddLabelToComment adds a label to the tracking comment
func AddLabelToComment(spc SCMClient, org, repo string, number int, label string) error {
	comment, err := GetLabelReportComment(spc, org, repo, number)
	if err != nil {
		return err
	}
	existingLabels := parseLabelReportComment(comment)

	existingLabels.Insert(label)

	newCommentBody := CreateLabelComment(existingLabels.List())

	if comment == nil {
		return spc.CreateComment(org, repo, number, true, newCommentBody)
	}
	if comment.Body != newCommentBody {
		return spc.EditComment(org, repo, number, comment.ID, newCommentBody, true)
	}
	return nil
}

func parseLabelReportComment(comment *scm.Comment) sets.String {
	labels := sets.NewString()
	if comment == nil {
		return labels
	}
	var tracking bool
	for _, line := range strings.Split(comment.Body, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "|")
		line = strings.TrimSuffix(line, "|")
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "---") {
			tracking = true
		} else if len(line) == 0 {
			tracking = false
		} else if tracking {
			labels.Insert(line)
		}
	}
	return labels
}

// GetLabelReportComment gets the scm.Comment, if present, containing the label report.
func GetLabelReportComment(spc SCMClient, org, repo string, number int) (*scm.Comment, error) {
	prcs, err := spc.ListPullRequestComments(org, repo, number)
	if err != nil {
		return nil, fmt.Errorf("error listing comments: %v", err)
	}
	botName, err := spc.BotName()
	if err != nil {
		return nil, fmt.Errorf("error getting bot name: %v", err)
	}

	for _, comment := range prcs {
		if comment.Author.Login != botName {
			continue
		}
		if !strings.Contains(comment.Body, labelTag) {
			continue
		}

		return comment, nil
	}
	return nil, nil
}

// CreateLabelComment creates the appropriate comment structure for the relevant labels
func CreateLabelComment(labels []string) string {
	lines := []string{
		"The following labels have been applied to this pull request:",
		"",
		"| Label name |",
		"| --- |",
	}
	for _, l := range labels {
		lines = append(lines, fmt.Sprintf("| %s |", l))
	}
	lines = append(lines, []string{
		"",
		"<" + labelTag + ">",
	}...)
	return strings.Join(lines, "\n")
}
