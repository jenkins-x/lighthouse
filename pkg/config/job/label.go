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

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/validation"
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
	// CreatedByLighthouseLabel is added on resources created by Lighthosue.
	// Since resources often live in another cluster/namespace,
	// the k8s garbage collector would immediately delete these
	// resources
	CreatedByLighthouseLabel = "created-by-lighthouse"
)

// Labels returns a string slice with label consts from kube.
func Labels() []string {
	return []string{LighthouseJobTypeLabel, CreatedByLighthouseLabel, LighthouseJobIDLabel}
}

// ValidateLabels validates labels (not using reserved labels, valid names and valid values)
func ValidateLabels(labels map[string]string) error {
	for label, value := range labels {
		for _, prowLabel := range Labels() {
			if label == prowLabel {
				return fmt.Errorf("label %s is reserved for decoration", label)
			}
		}
		if errs := validation.IsQualifiedName(label); len(errs) != 0 {
			return fmt.Errorf("invalid label %s: %v", label, errs)
		}
		if errs := validation.IsValidLabelValue(labels[label]); len(errs) != 0 {
			return fmt.Errorf("label %s has invalid value %s: %v", label, value, errs)
		}
	}
	return nil
}
