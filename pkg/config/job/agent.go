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

package job

// Various agents.
const (
	// JenkinsXAgent is the agent type for running Jenkins X pipelines
	JenkinsXAgent = "jenkins-x"

	// LegacyDefaultAgent is a backwards compatible way of dealing with legacy cases of "tekton" as the default agent, but meaning Jenkins X
	LegacyDefaultAgent = "tekton"

	// TektonPipelineAgent is the agent type for running Tekton Pipeline pipelines
	TektonPipelineAgent = "tekton-pipeline"

	// JenkinsAgent is the agent type for running Jenkins pipelines
	JenkinsAgent = "jenkins"
)

// AvailablePipelineAgentTypes returns a slice of all available pipeline agent types
func AvailablePipelineAgentTypes() []string {
	return []string{JenkinsXAgent, LegacyDefaultAgent, TektonPipelineAgent, JenkinsAgent}
}
