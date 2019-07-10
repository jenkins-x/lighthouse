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

// Package pjutil contains helpers for working with PlumberJobs.
package pjutil

import (
	"bytes"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/plumber"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/test-infra/prow/kube"

	"github.com/jenkins-x/lighthouse/pkg/prow/config"
	"github.com/jenkins-x/lighthouse/pkg/prow/github"
)

// NewPlumberJob initializes a PlumberJob out of a PlumberJobSpec.
func NewPlumberJob(spec plumber.PlumberJobSpec, extraLabels, extraAnnotations map[string]string) plumber.PlumberJob {
	labels, annotations := LabelsAndAnnotationsForSpec(spec, extraLabels, extraAnnotations)
	newID, _ := uuid.NewV1()

	return plumber.PlumberJob{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "prow.k8s.io/v1",
			Kind:       "PlumberJob",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        newID.String(),
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: spec,
		Status: plumber.PlumberJobStatus{
			StartTime: metav1.Now(),
			State:     plumber.TriggeredState,
		},
	}
}

func createRefs(pr *scm.PullRequest, baseSHA string) plumber.Refs {
	org := pr.Base.Repo.Namespace
	repo := pr.Base.Repo.Name
	number := pr.Number
	return plumber.Refs{
		Org:  org,
		Repo: repo,
		// TODO
		/*
			RepoLink: pr.Base.Repo.Link,
			BaseLink: fmt.Sprintf("%s/commit/%s", repoLink, baseSHA),
		*/
		BaseRef: pr.Base.Ref,
		BaseSHA: baseSHA,
		Pulls: []plumber.Pull{
			{
				Number: number,
				Author: pr.Author.Login,
				SHA:    pr.Head.Sha,
				// TODO
				/*
					Link:       pr.Link,
					AuthorLink: pr.Author.Link,
					CommitLink: fmt.Sprintf("%s/pull/%d/commits/%s", repoLink, number, pr.Head.Sha),
				*/
			},
		},
	}
}

// NewPresubmit converts a config.Presubmit into a builder.PlumberJob.
// The builder.Refs are configured correctly per the pr, baseSHA.
// The eventGUID becomes a github.EventGUID label.
func NewPresubmit(pr *scm.PullRequest, baseSHA string, job config.Presubmit, eventGUID string) plumber.PlumberJob {
	refs := createRefs(pr, baseSHA)
	labels := make(map[string]string)
	for k, v := range job.Labels {
		labels[k] = v
	}
	annotations := make(map[string]string)
	for k, v := range job.Annotations {
		annotations[k] = v
	}
	labels[github.EventGUID] = eventGUID
	return NewPlumberJob(PresubmitSpec(job, refs), labels, annotations)
}

// PresubmitSpec initializes a PlumberJobSpec for a given presubmit job.
func PresubmitSpec(p config.Presubmit, refs plumber.Refs) plumber.PlumberJobSpec {
	pjs := specFromJobBase(p.JobBase)
	pjs.Type = plumber.PresubmitJob
	pjs.Context = p.Context
	pjs.Report = !p.SkipReport
	pjs.RerunCommand = p.RerunCommand
	pjs.Refs = completePrimaryRefs(refs, p.JobBase)

	return pjs
}

// PostsubmitSpec initializes a PlumberJobSpec for a given postsubmit job.
func PostsubmitSpec(p config.Postsubmit, refs plumber.Refs) plumber.PlumberJobSpec {
	pjs := specFromJobBase(p.JobBase)
	pjs.Type = plumber.PostsubmitJob
	pjs.Context = p.Context
	pjs.Report = !p.SkipReport
	pjs.Refs = completePrimaryRefs(refs, p.JobBase)

	return pjs
}

// PeriodicSpec initializes a PlumberJobSpec for a given periodic job.
func PeriodicSpec(p config.Periodic) plumber.PlumberJobSpec {
	pjs := specFromJobBase(p.JobBase)
	pjs.Type = plumber.PeriodicJob

	return pjs
}

// BatchSpec initializes a PlumberJobSpec for a given batch job and ref spec.
func BatchSpec(p config.Presubmit, refs plumber.Refs) plumber.PlumberJobSpec {
	pjs := specFromJobBase(p.JobBase)
	pjs.Type = plumber.BatchJob
	pjs.Context = p.Context
	pjs.Refs = completePrimaryRefs(refs, p.JobBase)

	return pjs
}

func specFromJobBase(jb config.JobBase) plumber.PlumberJobSpec {
	var namespace string
	if jb.Namespace != nil {
		namespace = *jb.Namespace
	}
	return plumber.PlumberJobSpec{
		Job:              jb.Name,
		Cluster:          jb.Cluster,
		Namespace:        namespace,
		MaxConcurrency:   jb.MaxConcurrency,
		ErrorOnEviction:  jb.ErrorOnEviction,
		DecorationConfig: jb.DecorationConfig,

		/*		ExtraRefs:        jb.ExtraRefs,

				PodSpec:   jb.Spec,
				BuildSpec: jb.BuildSpec,
		*/
	}
}

func completePrimaryRefs(refs plumber.Refs, jb config.JobBase) *plumber.Refs {
	if jb.PathAlias != "" {
		refs.PathAlias = jb.PathAlias
	}
	if jb.CloneURI != "" {
		refs.CloneURI = jb.CloneURI
	}
	refs.SkipSubmodules = jb.SkipSubmodules
	// TODO
	//refs.CloneDepth = jb.CloneDepth
	return &refs
}

// PartitionActive separates the provided plumberJobs into pending and triggered
// and returns them inside channels so that they can be consumed in parallel
// by different goroutines. Complete plumberJobs are filtered out. Controller
// loops need to handle pending jobs first so they can conform to maximum
// concurrency requirements that different jobs may have.
func PartitionActive(pjs []plumber.PlumberJob) (pending, triggered chan plumber.PlumberJob) {
	// Size channels correctly.
	pendingCount, triggeredCount := 0, 0
	for _, pj := range pjs {
		switch pj.Status.State {
		case plumber.PendingState:
			pendingCount++
		case plumber.TriggeredState:
			triggeredCount++
		}
	}
	pending = make(chan plumber.PlumberJob, pendingCount)
	triggered = make(chan plumber.PlumberJob, triggeredCount)

	// Partition the jobs into the two separate channels.
	for _, pj := range pjs {
		switch pj.Status.State {
		case plumber.PendingState:
			pending <- pj
		case plumber.TriggeredState:
			triggered <- pj
		}
	}
	close(pending)
	close(triggered)
	return pending, triggered
}

// GetLatestPlumberJobs filters through the provided plumberJobs and returns
// a map of jobType jobs to their latest plumberJobs.
func GetLatestPlumberJobs(pjs []plumber.PlumberJob, jobType plumber.PlumberJobType) map[string]plumber.PlumberJob {
	latestJobs := make(map[string]plumber.PlumberJob)
	for _, j := range pjs {
		if j.Spec.Type != jobType {
			continue
		}
		name := j.Spec.Job
		if j.Status.StartTime.After(latestJobs[name].Status.StartTime.Time) {
			latestJobs[name] = j
		}
	}
	return latestJobs
}

// PlumberJobFields extracts logrus fields from a plumberJob useful for logging.
func PlumberJobFields(pj *plumber.PlumberJob) logrus.Fields {
	fields := make(logrus.Fields)
	fields["name"] = pj.ObjectMeta.Name
	fields["job"] = pj.Spec.Job
	fields["type"] = pj.Spec.Type
	if len(pj.ObjectMeta.Labels[github.EventGUID]) > 0 {
		fields[github.EventGUID] = pj.ObjectMeta.Labels[github.EventGUID]
	}
	if pj.Spec.Refs != nil && len(pj.Spec.Refs.Pulls) == 1 {
		fields[github.PrLogField] = pj.Spec.Refs.Pulls[0].Number
		fields[github.RepoLogField] = pj.Spec.Refs.Repo
		fields[github.OrgLogField] = pj.Spec.Refs.Org
	}
	return fields
}

// JobURL returns the expected URL for PlumberJobStatus.
//
// TODO(fejta): consider moving default JobURLTemplate and JobURLPrefix out of plank
func JobURL(plank config.Plank, pj plumber.PlumberJob, log *logrus.Entry) string {
	/*	if pj.Spec.DecorationConfig != nil && plank.GetJobURLPrefix(pj.Spec.Refs) != "" {
			spec := downwardapi.NewJobSpec(pj.Spec, pj.Status.BuildID, pj.Name)
			gcsConfig := pj.Spec.DecorationConfig.GCSConfiguration
			_, gcsPath, _ := gcsupload.PathsForJob(gcsConfig, &spec, "")

			prefix, _ := url.Parse(plank.GetJobURLPrefix(pj.Spec.Refs))
			prefix.Path = path.Join(prefix.Path, gcsConfig.Bucket, gcsPath)
			return prefix.String()
		}
	*/
	var b bytes.Buffer
	if err := plank.JobURLTemplate.Execute(&b, &pj); err != nil {
		log.WithFields(PlumberJobFields(&pj)).Errorf("error executing URL template: %v", err)
	} else {
		return b.String()
	}
	return ""
}

// LabelsAndAnnotationsForSpec returns a minimal set of labels to add to plumberJobs or its owned resources.
//
// User-provided extraLabels and extraAnnotations values will take precedence over auto-provided values.
func LabelsAndAnnotationsForSpec(spec plumber.PlumberJobSpec, extraLabels, extraAnnotations map[string]string) (map[string]string, map[string]string) {
	jobNameForLabel := spec.Job
	if len(jobNameForLabel) > validation.LabelValueMaxLength {
		// TODO(fejta): consider truncating middle rather than end.
		jobNameForLabel = strings.TrimRight(spec.Job[:validation.LabelValueMaxLength], ".-")
		logrus.WithFields(logrus.Fields{
			"job":       spec.Job,
			"key":       plumber.PlumberJobAnnotation,
			"value":     spec.Job,
			"truncated": jobNameForLabel,
		}).Info("Cannot use full job name, will truncate.")
	}
	labels := map[string]string{
		kube.CreatedByProw:           "true",
		plumber.PlumberJobTypeLabel:  string(spec.Type),
		plumber.PlumberJobAnnotation: jobNameForLabel,
	}
	if spec.Type != plumber.PeriodicJob && spec.Refs != nil {
		labels[kube.OrgLabel] = spec.Refs.Org
		labels[kube.RepoLabel] = spec.Refs.Repo
		if len(spec.Refs.Pulls) > 0 {
			labels[kube.PullLabel] = strconv.Itoa(spec.Refs.Pulls[0].Number)
		}
	}

	for k, v := range extraLabels {
		labels[k] = v
	}

	// let's validate labels
	for key, value := range labels {
		if errs := validation.IsValidLabelValue(value); len(errs) > 0 {
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
		plumber.PlumberJobAnnotation: spec.Job,
	}
	for k, v := range extraAnnotations {
		annotations[k] = v
	}

	return labels, annotations
}

// LabelsAndAnnotationsForJob returns a standard set of labels to add to pod/build/etc resources.
func LabelsAndAnnotationsForJob(pj plumber.PlumberJob) (map[string]string, map[string]string) {
	var extraLabels map[string]string
	if extraLabels = pj.ObjectMeta.Labels; extraLabels == nil {
		extraLabels = map[string]string{}
	}
	extraLabels[plumber.PlumberJobIDLabel] = pj.ObjectMeta.Name
	return LabelsAndAnnotationsForSpec(pj.Spec, extraLabels, nil)
}
