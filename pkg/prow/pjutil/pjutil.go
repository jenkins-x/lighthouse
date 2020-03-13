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

// Package pjutil contains helpers for working with LighthouseJobs.
package pjutil

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/launcher"
	"github.com/jenkins-x/lighthouse/pkg/prow/config"
	"github.com/jenkins-x/lighthouse/pkg/prow/gitprovider"
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
			APIVersion: "prow.k8s.io/v1",
			Kind:       "PipelineOptions",
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
	return v1alpha1.Refs{
		Org:      org,
		Repo:     repo,
		RepoLink: repoLink,
		BaseLink: fmt.Sprintf("%s/commit/%s", repoLink, baseSHA),

		BaseRef: pr.Base.Ref,
		BaseSHA: baseSHA,
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
	labels[gitprovider.EventGUID] = eventGUID
	return NewLighthouseJob(PresubmitSpec(job, refs), labels, annotations)
}

// PresubmitSpec initializes a PipelineOptionsSpec for a given presubmit job.
func PresubmitSpec(p config.Presubmit, refs v1alpha1.Refs) v1alpha1.LighthouseJobSpec {
	pjs := specFromJobBase(p.JobBase)
	pjs.Type = v1alpha1.PresubmitJob
	pjs.Context = p.Context
	pjs.RerunCommand = p.RerunCommand
	pjs.Refs = completePrimaryRefs(refs, p.JobBase)

	return pjs
}

// PostsubmitSpec initializes a PipelineOptionsSpec for a given postsubmit job.
func PostsubmitSpec(p config.Postsubmit, refs v1alpha1.Refs) v1alpha1.LighthouseJobSpec {
	pjs := specFromJobBase(p.JobBase)
	pjs.Type = v1alpha1.PostsubmitJob
	pjs.Context = p.Context
	pjs.Refs = completePrimaryRefs(refs, p.JobBase)

	return pjs
}

// PeriodicSpec initializes a PipelineOptionsSpec for a given periodic job.
func PeriodicSpec(p config.Periodic) v1alpha1.LighthouseJobSpec {
	pjs := specFromJobBase(p.JobBase)
	pjs.Type = v1alpha1.PeriodicJob

	return pjs
}

// BatchSpec initializes a PipelineOptionsSpec for a given batch job and ref spec.
func BatchSpec(p config.Presubmit, refs v1alpha1.Refs) v1alpha1.LighthouseJobSpec {
	pjs := specFromJobBase(p.JobBase)
	pjs.Type = v1alpha1.BatchJob
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
		Job:            jb.Name,
		Namespace:      namespace,
		MaxConcurrency: jb.MaxConcurrency,
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
	if len(pj.ObjectMeta.Labels[gitprovider.EventGUID]) > 0 {
		fields[gitprovider.EventGUID] = pj.ObjectMeta.Labels[gitprovider.EventGUID]
	}
	if pj.Spec.Refs != nil && len(pj.Spec.Refs.Pulls) == 1 {
		fields[gitprovider.PrLogField] = pj.Spec.Refs.Pulls[0].Number
		fields[gitprovider.RepoLogField] = pj.Spec.Refs.Repo
		fields[gitprovider.OrgLogField] = pj.Spec.Refs.Org
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
	if len(jobNameForLabel) > validation.LabelValueMaxLength {
		// TODO(fejta): consider truncating middle rather than end.
		jobNameForLabel = strings.TrimRight(spec.Job[:validation.LabelValueMaxLength], ".-")
		logrus.WithFields(logrus.Fields{
			"job":       spec.Job,
			"key":       launcher.LighthouseJobAnnotation,
			"value":     spec.Job,
			"truncated": jobNameForLabel,
		}).Info("Cannot use full job name, will truncate.")
	}
	labels := map[string]string{
		launcher.CreatedByLighthouse:               "true",
		launcher.LighthouseJobTypeLabel:  string(spec.Type),
		launcher.LighthouseJobAnnotation: jobNameForLabel,
	}
	if spec.Type != v1alpha1.PeriodicJob && spec.Refs != nil {
		labels[launcher.OrgLabel] = spec.Refs.Org
		labels[launcher.RepoLabel] = spec.Refs.Repo
		if len(spec.Refs.Pulls) > 0 {
			labels[launcher.PullLabel] = strconv.Itoa(spec.Refs.Pulls[0].Number)
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
		launcher.LighthouseJobAnnotation: spec.Job,
	}
	for k, v := range extraAnnotations {
		annotations[k] = v
	}

	return labels, annotations
}

// LabelsAndAnnotationsForJob returns a standard set of labels to add to pod/build/etc resources.
func LabelsAndAnnotationsForJob(pj v1alpha1.LighthouseJob) (map[string]string, map[string]string) {
	var extraLabels map[string]string
	if extraLabels = pj.ObjectMeta.Labels; extraLabels == nil {
		extraLabels = map[string]string{}
	}
	extraLabels[launcher.LighthouseJobIDLabel] = pj.ObjectMeta.Name
	return LabelsAndAnnotationsForSpec(pj.Spec, extraLabels, nil)
}
