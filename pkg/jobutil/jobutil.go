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
	"bytes"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse-config/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/jenkins-x/lighthouse/pkg/util"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

// NewLighthouseJob initializes a LighthouseJob out of a LighthouseJobSpec.
func NewLighthouseJob(spec v1alpha1.LighthouseJobSpec, extraLabels, extraAnnotations map[string]string) v1alpha1.LighthouseJob {
	labels, annotations := LabelsAndAnnotationsForSpec(spec, extraLabels, extraAnnotations)
	newID, _ := uuid.NewV1()

	return v1alpha1.LighthouseJob{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "lighthouse.jenkins.io/v1alpha1",
			Kind:       "LighthouseJob",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        newID.String(),
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: spec,
	}
}

func createRefs(pr *scm.PullRequest, baseSHA string) v1alpha1.Refs {
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
			},
		},
	}
}

// NewPresubmit converts a config.Presubmit into a builder.PipelineOptions.
// The builder.Refs are configured correctly per the pr, baseSHA.
// The eventGUID becomes a gitprovider.EventGUID label.
func NewPresubmit(pr *scm.PullRequest, baseSHA string, job config.Presubmit, eventGUID string) v1alpha1.LighthouseJob {
	refs := createRefs(pr, baseSHA)
	labels := make(map[string]string)
	for k, v := range job.Labels {
		labels[k] = v
	}
	annotations := make(map[string]string)
	for k, v := range job.Annotations {
		annotations[k] = v
	}
	labels[scmprovider.EventGUID] = eventGUID
	return NewLighthouseJob(PresubmitSpec(job, refs), labels, annotations)
}

// PresubmitSpec initializes a PipelineOptionsSpec for a given presubmit job.
func PresubmitSpec(p config.Presubmit, refs v1alpha1.Refs) v1alpha1.LighthouseJobSpec {
	pjs := specFromJobBase(p.JobBase)
	pjs.Type = config.PresubmitJob
	pjs.Context = p.Context
	pjs.RerunCommand = p.RerunCommand
	pjs.Refs = completePrimaryRefs(refs, p.JobBase)

	return pjs
}

// PostsubmitSpec initializes a PipelineOptionsSpec for a given postsubmit job.
func PostsubmitSpec(p config.Postsubmit, refs v1alpha1.Refs) v1alpha1.LighthouseJobSpec {
	pjs := specFromJobBase(p.JobBase)
	pjs.Type = config.PostsubmitJob
	pjs.Context = p.Context
	pjs.Refs = completePrimaryRefs(refs, p.JobBase)

	return pjs
}

// PeriodicSpec initializes a PipelineOptionsSpec for a given periodic job.
func PeriodicSpec(p config.Periodic) v1alpha1.LighthouseJobSpec {
	pjs := specFromJobBase(p.JobBase)
	pjs.Type = config.PeriodicJob

	return pjs
}

// BatchSpec initializes a PipelineOptionsSpec for a given batch job and ref spec.
func BatchSpec(p config.Presubmit, refs v1alpha1.Refs) v1alpha1.LighthouseJobSpec {
	pjs := specFromJobBase(p.JobBase)
	pjs.Type = config.BatchJob
	pjs.Context = p.Context
	pjs.Refs = completePrimaryRefs(refs, p.JobBase)

	return pjs
}

func specFromJobBase(jb config.JobBase) v1alpha1.LighthouseJobSpec {
	var namespace string
	if jb.Namespace != nil {
		namespace = *jb.Namespace
	}
	return v1alpha1.LighthouseJobSpec{
		Agent:           jb.Agent,
		Job:             jb.Name,
		Namespace:       namespace,
		MaxConcurrency:  jb.MaxConcurrency,
		PodSpec:         jb.Spec,
		PipelineRunSpec: jb.PipelineRunSpec,
	}
}

func completePrimaryRefs(refs v1alpha1.Refs, jb config.JobBase) *v1alpha1.Refs {
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

// LighthouseJobFields extracts logrus fields from a LighthouseJob useful for logging.
func LighthouseJobFields(pj *v1alpha1.LighthouseJob) logrus.Fields {
	fields := make(logrus.Fields)
	fields["name"] = pj.ObjectMeta.Name
	fields["job"] = pj.Spec.Job
	fields["type"] = pj.Spec.Type
	if len(pj.ObjectMeta.Labels[scmprovider.EventGUID]) > 0 {
		fields[scmprovider.EventGUID] = pj.ObjectMeta.Labels[scmprovider.EventGUID]
	}
	if pj.Spec.Refs != nil && len(pj.Spec.Refs.Pulls) == 1 {
		fields[scmprovider.PrLogField] = pj.Spec.Refs.Pulls[0].Number
		fields[scmprovider.RepoLogField] = pj.Spec.Refs.Repo
		fields[scmprovider.OrgLogField] = pj.Spec.Refs.Org
	}
	return fields
}

// JobURL returns the expected URL for LighthouseJobStatus.
//
// TODO(fejta): consider moving default JobURLTemplate and JobURLPrefix out of plank
func JobURL(plank config.Plank, pj v1alpha1.LighthouseJob, log *logrus.Entry) string {
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
		log.WithFields(LighthouseJobFields(&pj)).Errorf("error executing URL template: %v", err)
	} else {
		return b.String()
	}
	return ""
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
		config.CreatedByLighthouse:    "true",
		config.LighthouseJobTypeLabel: string(spec.Type),
		util.LighthouseJobAnnotation:  jobNameForLabel,
	}
	if contextNameForLabel != "" {
		labels[util.ContextLabel] = contextNameForLabel
	}
	if spec.Type != config.PeriodicJob && spec.Refs != nil {
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
	extraLabels[config.LighthouseJobIDLabel] = lj.ObjectMeta.Name
	if buildID != "" {
		extraLabels[util.BuildNumLabel] = buildID
	}
	return LabelsAndAnnotationsForSpec(lj.Spec, extraLabels, nil)
}
