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

package job

// // RetestPresubmits returns all presubmits that should be run given a /retest command.
// // This is the set of all presubmits intersected with ((alwaysRun + runContexts) - skipContexts)
// func (c *JobConfig) RetestPresubmits(fullRepoName string, skipContexts, runContexts sets.String) []Presubmit {
// 	var result []Presubmit
// 	if jobs, ok := c.Presubmits[fullRepoName]; ok {
// 		for _, job := range jobs {
// 			if skipContexts.Has(job.Context) {
// 				continue
// 			}
// 			if job.AlwaysRun || job.RunIfChanged != "" || runContexts.Has(job.Context) {
// 				result = append(result, job)
// 			}
// 		}
// 	}
// 	return result
// }

// // GetPresubmit returns the presubmit job for the provided repo and job name.
// func (c *JobConfig) GetPresubmit(repo, jobName string) *Presubmit {
// 	presubmits := c.AllPresubmits([]string{repo})
// 	for i := range presubmits {
// 		ps := presubmits[i]
// 		if ps.Name == jobName {
// 			return &ps
// 		}
// 	}
// 	return nil
// }
