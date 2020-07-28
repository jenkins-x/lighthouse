/*
 * The MIT License
 *
 * Copyright (c) 2020, CloudBees, Inc.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 */

package config

// PullRequestMergeType inidicates the type of the pull request
type PullRequestMergeType string

// Possible types of merges for the GitHub merge API
const (
	MergeMerge  PullRequestMergeType = "merge"
	MergeRebase PullRequestMergeType = "rebase"
	MergeSquash PullRequestMergeType = "squash"
)

// PipelineKind specifies how the job is triggered.
type PipelineKind string

// Various job types.
const (
	// PresubmitJob means it runs on unmerged PRs.
	PresubmitJob PipelineKind = "presubmit"
	// PostsubmitJob means it runs on each new commit.
	PostsubmitJob PipelineKind = "postsubmit"
	// Periodic job means it runs on a time-basis, unrelated to git changes.
	PeriodicJob PipelineKind = "periodic"
	// BatchJob tests multiple unmerged PRs at the same time.
	BatchJob PipelineKind = "batch"
)

const (
	// LighthouseJobTypeLabel is added in resources created by lighthouse and
	// carries the job type (presubmit, postsubmit, periodic, batch)
	// that the pod is running.
	LighthouseJobTypeLabel = "lighthouse.jenkins-x.io/type"

	// LighthouseJobIDLabel is added in resources created by lighthouse and
	// carries the ID of the LighthouseJob that the pod is fulfilling.
	// We also name resources after the LighthouseJob that spawned them but
	// this allows for multiple resources to be linked to one
	// LighthouseJob.
	LighthouseJobIDLabel = "lighthouse.jenkins-x.io/id"

	// CreatedByLighthouse is added on resources created by Lighthosue.
	// Since resources often live in another cluster/namespace,
	// the k8s garbage collector would immediately delete these
	// resources
	CreatedByLighthouse = "created-by-lighthouse"

	// DefaultClusterAlias specifies the default context for resources owned by jobs (pods/builds).
	DefaultClusterAlias = "default"

	// JenkinsXAgent is the agent type for running Jenkins X pipelines
	JenkinsXAgent = "jenkins-x"

	// LegacyDefaultAgent is a backwards compatible way of dealing with legacy cases of "tekton" as the default agent, but meaning Jenkins X
	LegacyDefaultAgent = "tekton"

	// TektonPipelineAgent is the agent type for running Tekton Pipeline pipelines
	TektonPipelineAgent = "tekton-pipeline"
)

// AvailablePipelineAgentTypes returns a slice of all available pipeline agent types
func AvailablePipelineAgentTypes() []string {
	return []string{JenkinsXAgent, LegacyDefaultAgent, TektonPipelineAgent}
}
