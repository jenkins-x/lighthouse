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

// Package jobutil contains helpers for working with LighthouseJobs.
package jobutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/types"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	maxGenerateNamePrefix = 32
)

// lighthouseClient a minimalistic lighthouse client required by the aborter
type lighthouseClient interface {
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.LighthouseJob, err error)
}

// NewLighthouseJob initializes a LighthouseJob out of a LighthouseJobSpec.
func NewLighthouseJob(spec v1alpha1.LighthouseJobSpec, extraLabels, extraAnnotations map[string]string) v1alpha1.LighthouseJob {
	labels, annotations := LabelsAndAnnotationsForSpec(spec, extraLabels, extraAnnotations)

	generateName := GenerateName(&spec)
	return v1alpha1.LighthouseJob{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "lighthouse.jenkins.io/v1alpha1",
			Kind:       "LighthouseJob",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: generateName,
			Labels:       labels,
			Annotations:  annotations,
		},
		Spec: spec,
	}
}

func createRefs(pr *scm.PullRequest, baseSHA string, prRefFmt string) v1alpha1.Refs {
	org := pr.Base.Repo.Namespace
	repo := pr.Base.Repo.Name
	number := pr.Number
	repoLink := pr.Base.Repo.Link
	cloneURL := pr.Base.Repo.Clone
	return v1alpha1.Refs{
		Org:      org,
		Repo:     repo,
		RepoLink: repoLink,
		BaseLink: fmt.Sprintf("%s/commit/%s", repoLink, baseSHA),

		BaseRef:  pr.Base.Ref,
		BaseSHA:  baseSHA,
		CloneURI: cloneURL,
		Pulls: []v1alpha1.Pull{
			{
				Number:     number,
				Author:     pr.Author.Login,
				SHA:        pr.Head.Sha,
				Link:       pr.Link,
				AuthorLink: pr.Author.Link,
				CommitLink: fmt.Sprintf("%s/pull/%d/commits/%s", repoLink, number, pr.Head.Sha),
				Ref:        fmt.Sprintf(prRefFmt, number),
			},
		},
	}
}

// NewPresubmit converts a config.Presubmit into a builder.PipelineOptions.
// The builder.Refs are configured correctly per the pr, baseSHA.
// The eventGUID becomes a gitprovider.EventGUID label.
func NewPresubmit(logger *logrus.Entry, pr *scm.PullRequest, baseSHA string, job job.Presubmit, eventGUID string, prRefFmt string) v1alpha1.LighthouseJob {
	refs := createRefs(pr, baseSHA, prRefFmt)
	labels := make(map[string]string)
	for k, v := range job.Labels {
		labels[k] = v
	}
	annotations := make(map[string]string)
	for k, v := range job.Annotations {
		annotations[k] = v
	}
	labels[scmprovider.EventGUID] = eventGUID
	return NewLighthouseJob(PresubmitSpec(logger, job, refs), labels, annotations)
}

// PresubmitSpec initializes a PipelineOptionsSpec for a given presubmit job.
func PresubmitSpec(logger *logrus.Entry, p job.Presubmit, refs v1alpha1.Refs) v1alpha1.LighthouseJobSpec {
	pjs := specFromJobBase(logger, p.Base)
	pjs.Type = job.PresubmitJob
	pjs.Context = p.Context
	pjs.RerunCommand = p.RerunCommand
	pjs.Refs = completePrimaryRefs(refs, p.Base)

	if p.JenkinsSpec != nil {
		pjs.JenkinsSpec = &v1alpha1.JenkinsSpec{
			BranchSourceJob: p.JenkinsSpec.BranchSourceJob,
		}
	}

	return pjs
}

// PostsubmitSpec initializes a PipelineOptionsSpec for a given postsubmit job.
func PostsubmitSpec(logger *logrus.Entry, p job.Postsubmit, refs v1alpha1.Refs) v1alpha1.LighthouseJobSpec {
	pjs := specFromJobBase(logger, p.Base)
	pjs.Type = job.PostsubmitJob
	pjs.Context = p.Context
	pjs.Refs = completePrimaryRefs(refs, p.Base)

	if p.JenkinsSpec != nil {
		pjs.JenkinsSpec = &v1alpha1.JenkinsSpec{
			BranchSourceJob: p.JenkinsSpec.BranchSourceJob,
		}
	}

	return pjs
}

// DeploymentSpec initializes a PipelineOptionsSpec for a given deployment job.
func DeploymentSpec(logger *logrus.Entry, p job.Deployment, refs v1alpha1.Refs) v1alpha1.LighthouseJobSpec {
	pjs := specFromJobBase(logger, p.Base)
	pjs.Type = job.DeploymentJob
	pjs.Context = p.Context
	pjs.Refs = completePrimaryRefs(refs, p.Base)

	return pjs
}

// PeriodicSpec initializes a PipelineOptionsSpec for a given periodic job.
func PeriodicSpec(logger *logrus.Entry, p job.Periodic, refs v1alpha1.Refs) v1alpha1.LighthouseJobSpec {
	pjs := specFromJobBase(logger, p.Base)
	pjs.Type = job.PeriodicJob
	pjs.Context = p.Context
	pjs.Refs = completePrimaryRefs(refs, p.Base)

	return pjs
}

// BatchSpec initializes a PipelineOptionsSpec for a given batch job and ref spec.
func BatchSpec(logger *logrus.Entry, p job.Presubmit, refs v1alpha1.Refs) v1alpha1.LighthouseJobSpec {
	pjs := specFromJobBase(logger, p.Base)
	pjs.Type = job.BatchJob
	pjs.Context = p.Context
	pjs.Refs = completePrimaryRefs(refs, p.Base)

	return pjs
}

func specFromJobBase(logger *logrus.Entry, jb job.Base) v1alpha1.LighthouseJobSpec {
	// if we have not yet loaded the PipelineRunSpec then lets do it now
	if jb.PipelineRunSpec == nil {
		logger = logger.WithField("JobName", jb.Name)
		err := jb.LoadPipeline(logger)
		if err != nil {
			logger.WithError(err).Warn("failed to lazy load the PipelineRunSpec")
		}
	}
	var namespace string
	if jb.Namespace != nil {
		namespace = *jb.Namespace
	}
	return v1alpha1.LighthouseJobSpec{
		Agent:             jb.Agent,
		Job:               jb.Name,
		Namespace:         namespace,
		MaxConcurrency:    jb.MaxConcurrency,
		PodSpec:           jb.Spec,
		PipelineRunSpec:   jb.PipelineRunSpec,
		PipelineRunParams: jb.PipelineRunParams,
	}
}

func completePrimaryRefs(refs v1alpha1.Refs, jb job.Base) *v1alpha1.Refs {
	if jb.PathAlias != "" {
		refs.PathAlias = jb.PathAlias
	}
	if jb.CloneURI != "" {
		refs.CloneURI = jb.CloneURI
	}
	refs.SkipSubmodules = jb.SkipSubmodules
	// TODO
	// refs.CloneDepth = jb.CloneDepth
	return &refs
}

// GenerateName generates a meaningful name for the LighthouseJob from the spec
func GenerateName(spec *v1alpha1.LighthouseJobSpec) string {
	if spec.Refs == nil {
		return "missingref"
	}

	branch := spec.Refs.BaseRef
	if len(spec.Refs.Pulls) > 0 {
		branch = "pr-" + strconv.Itoa(spec.Refs.Pulls[0].Number)
	}
	name := addNonEmptyParts(spec.Refs.Org, spec.Refs.Repo, branch, spec.Context)
	name = util.ToValidName(name)
	if len(name) > maxGenerateNamePrefix {
		name = name[len(name)-maxGenerateNamePrefix:]
	}
	name = strings.TrimPrefix(name, "-")
	name = util.ToValidName(name)

	if !strings.HasSuffix(name, "-") {
		name += "-"
	}
	return name
}

func addNonEmptyParts(values ...string) string {
	var parts []string
	for _, v := range values {
		if v != "" {
			parts = append(parts, v)
		}
	}
	return strings.Join(parts, "-")
}

// LighthouseJobFields extracts logrus fields from a LighthouseJob useful for logging.
func LighthouseJobFields(lighthouseJob *v1alpha1.LighthouseJob) logrus.Fields {
	fields := make(logrus.Fields)
	fields["name"] = lighthouseJob.ObjectMeta.Name
	fields["job"] = lighthouseJob.Spec.Job
	fields["type"] = lighthouseJob.Spec.Type
	if len(lighthouseJob.ObjectMeta.Labels[scmprovider.EventGUID]) > 0 {
		fields[scmprovider.EventGUID] = lighthouseJob.ObjectMeta.Labels[scmprovider.EventGUID]
	}
	if lighthouseJob.Spec.Refs != nil && len(lighthouseJob.Spec.Refs.Pulls) == 1 {
		fields[scmprovider.PrLogField] = lighthouseJob.Spec.Refs.Pulls[0].Number
		fields[scmprovider.RepoLogField] = lighthouseJob.Spec.Refs.Repo
		fields[scmprovider.OrgLogField] = lighthouseJob.Spec.Refs.Org
	}

	if lighthouseJob.Spec.JenkinsSpec != nil {
		fields["github_based_job"] = lighthouseJob.Spec.JenkinsSpec.BranchSourceJob
	}
	return fields
}

// LabelsAndAnnotationsForSpec returns a minimal set of labels to add to LighthouseJobs or its owned resources.
//
// User-provided extraLabels and extraAnnotations values will take precedence over auto-provided values.
func LabelsAndAnnotationsForSpec(spec v1alpha1.LighthouseJobSpec, extraLabels, extraAnnotations map[string]string) (map[string]string, map[string]string) {
	jobNameForLabel := spec.Job
	contextNameForLabel := spec.Context
	if len(jobNameForLabel) > validation.LabelValueMaxLength {
		// TODO(fejta): consider truncating middle rather than end.
		jobNameForLabel = strings.TrimRight(spec.Job[:validation.LabelValueMaxLength], ".-")
		logrus.WithFields(logrus.Fields{
			"job":       spec.Job,
			"key":       util.LighthouseJobAnnotation,
			"value":     spec.Job,
			"truncated": jobNameForLabel,
		}).Info("Cannot use full job name, will truncate.")
	}
	if len(contextNameForLabel) > validation.LabelValueMaxLength {
		// TODO(fejta): consider truncating middle rather than end.
		contextNameForLabel = strings.TrimRight(spec.Context[:validation.LabelValueMaxLength], ".-")
		logrus.WithFields(logrus.Fields{
			"context":   spec.Context,
			"key":       util.ContextLabel,
			"value":     spec.Context,
			"truncated": contextNameForLabel,
		}).Info("Cannot use full context name, will truncate.")
	}
	labels := map[string]string{
		job.CreatedByLighthouseLabel: "true",
		job.LighthouseJobTypeLabel:   string(spec.Type),
		util.LighthouseJobAnnotation: jobNameForLabel,
	}
	if contextNameForLabel != "" {
		labels[util.ContextLabel] = contextNameForLabel
	}
	if spec.Type != job.PeriodicJob && spec.Refs != nil {
		labels[util.OrgLabel] = strings.ToLower(spec.Refs.Org)
		labels[util.RepoLabel] = spec.Refs.Repo
		labels[util.BranchLabel] = spec.GetBranch()
		labels[util.BaseSHALabel] = spec.Refs.BaseSHA
		if len(spec.Refs.Pulls) > 0 {
			labels[util.PullLabel] = strconv.Itoa(spec.Refs.Pulls[0].Number)
			labels[util.LastCommitSHALabel] = spec.Refs.Pulls[0].SHA
		} else {
			labels[util.LastCommitSHALabel] = spec.Refs.BaseSHA
		}
	}

	for k, v := range extraLabels {
		labels[k] = v
	}

	// let's validate labels
	for key, value := range labels {
		if errs := validation.IsValidLabelValue(value); len(errs) > 0 {
			// ToDo: Use util.GitKind function instead
			// For nested repos only in gitlab, we do not want to remove the sub group name, which comes before /
			if key == util.RepoLabel && os.Getenv("GIT_KIND") == "gitlab" {
				value = strings.Replace(value, "/", "-", -1)
			}
			// try to use basename of a path, if path contains invalid //
			base := filepath.Base(value)
			if errs := validation.IsValidLabelValue(base); len(errs) == 0 {
				labels[key] = base
				continue
			}
			logrus.WithFields(logrus.Fields{
				"key":    key,
				"value":  value,
				"errors": errs,
			}).Warn("Removing invalid label")
			delete(labels, key)
		}
	}

	annotations := map[string]string{
		util.LighthouseJobAnnotation: spec.Job,
	}
	if spec.Refs != nil && spec.Refs.CloneURI != "" {
		annotations[util.CloneURIAnnotation] = spec.Refs.CloneURI
	}
	for k, v := range extraAnnotations {
		annotations[k] = v
	}
	return labels, annotations
}

// LabelsAndAnnotationsForJob returns a standard set of labels to add to pod/build/etc resources.
func LabelsAndAnnotationsForJob(lj v1alpha1.LighthouseJob, buildID string) (map[string]string, map[string]string) {
	var extraLabels map[string]string
	if extraLabels = lj.ObjectMeta.Labels; extraLabels == nil {
		extraLabels = map[string]string{}
	}
	extraLabels[job.LighthouseJobIDLabel] = lj.ObjectMeta.Name
	if buildID != "" {
		extraLabels[util.BuildNumLabel] = buildID
	}

	var extraAnnotations map[string]string
	if extraAnnotations = lj.ObjectMeta.Annotations; extraAnnotations == nil {
		extraAnnotations = map[string]string{}
	}
	// ensure the opentelemetry annotations holding trace context
	// won't be copied to other resources
	delete(extraAnnotations, "lighthouse.jenkins-x.io/traceparent")
	delete(extraAnnotations, "lighthouse.jenkins-x.io/tracestate")
	return LabelsAndAnnotationsForSpec(lj.Spec, extraLabels, extraAnnotations)
}

// PartitionActive separates the provided prowjobs into pending and triggered
// and returns them inside channels so that they can be consumed in parallel
// by different goroutines. Complete prowjobs are filtered out. Controller
// loops need to handle pending jobs first so they can conform to maximum
// concurrency requirements that different jobs may have.
func PartitionActive(pjs []v1alpha1.LighthouseJob) (pending, triggered, aborted chan v1alpha1.LighthouseJob) {
	// Size channels correctly.
	pendingCount, triggeredCount, abortedCount := 0, 0, 0
	for _, pj := range pjs {
		switch pj.Status.State {
		case v1alpha1.PendingState:
			pendingCount++
		case v1alpha1.TriggeredState:
			triggeredCount++
		case v1alpha1.AbortedState:
			abortedCount++
		}
	}
	pending = make(chan v1alpha1.LighthouseJob, pendingCount)
	triggered = make(chan v1alpha1.LighthouseJob, triggeredCount)
	aborted = make(chan v1alpha1.LighthouseJob, abortedCount)

	// Partition the jobs into the two separate channels.
	for _, pj := range pjs {
		switch pj.Status.State {
		case v1alpha1.PendingState:
			pending <- pj
		case v1alpha1.TriggeredState:
			triggered <- pj
		case v1alpha1.AbortedState:
			if !pj.Complete() {
				aborted <- pj
			}
		}
	}
	close(pending)
	close(triggered)
	close(aborted)
	return pending, triggered, aborted
}
