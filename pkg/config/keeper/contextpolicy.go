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

package keeper

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"
)

// ContextPolicy configures options about how to handle various contexts.
type ContextPolicy struct {
	// whether to consider unknown contexts optional (skip) or required.
	SkipUnknownContexts       *bool    `json:"skip-unknown-contexts,omitempty"`
	RequiredContexts          []string `json:"required-contexts,omitempty"`
	RequiredIfPresentContexts []string `json:"required-if-present-contexts"`
	OptionalContexts          []string `json:"optional-contexts,omitempty"`
	// Infer required and optional jobs from Branch Protection configuration
	FromBranchProtection *bool `json:"from-branch-protection,omitempty"`
}

// Validate returns an error if any contexts are listed more than once in the config.
func (cp *ContextPolicy) Validate() error {
	if inter := sets.NewString(cp.RequiredContexts...).Intersection(sets.NewString(cp.OptionalContexts...)); inter.Len() > 0 {
		return fmt.Errorf("contexts %s are defined as required and optional", strings.Join(inter.List(), ", "))
	}
	if inter := sets.NewString(cp.RequiredContexts...).Intersection(sets.NewString(cp.RequiredIfPresentContexts...)); inter.Len() > 0 {
		return fmt.Errorf("contexts %s are defined as required and required if present", strings.Join(inter.List(), ", "))
	}
	if inter := sets.NewString(cp.OptionalContexts...).Intersection(sets.NewString(cp.RequiredIfPresentContexts...)); inter.Len() > 0 {
		return fmt.Errorf("contexts %s are defined as optional and required if present", strings.Join(inter.List(), ", "))
	}
	return nil
}

// Merge merges one ContextPolicy with another one
func (cp ContextPolicy) Merge(other ContextPolicy) ContextPolicy {
	mergeBool := func(a, b *bool) *bool {
		if b == nil {
			return a
		}
		return b
	}
	c := ContextPolicy{}
	c.FromBranchProtection = mergeBool(cp.FromBranchProtection, other.FromBranchProtection)
	c.SkipUnknownContexts = mergeBool(cp.SkipUnknownContexts, other.SkipUnknownContexts)
	required := sets.NewString(cp.RequiredContexts...)
	requiredIfPresent := sets.NewString(cp.RequiredIfPresentContexts...)
	optional := sets.NewString(cp.OptionalContexts...)
	required.Insert(other.RequiredContexts...)
	requiredIfPresent.Insert(other.RequiredIfPresentContexts...)
	optional.Insert(other.OptionalContexts...)
	if required.Len() > 0 {
		c.RequiredContexts = required.List()
	}
	if requiredIfPresent.Len() > 0 {
		c.RequiredIfPresentContexts = requiredIfPresent.List()
	}
	if optional.Len() > 0 {
		c.OptionalContexts = optional.List()
	}
	return c
}

// IsOptional checks whether a context can be ignored.
// Will return true if
// - context is registered as optional
// - required contexts are registered and the context provided is not required
// Will return false otherwise. Every context is required.
func (cp *ContextPolicy) IsOptional(c string) bool {
	if sets.NewString(cp.OptionalContexts...).Has(c) {
		return true
	}
	if sets.NewString(cp.RequiredContexts...).Has(c) {
		return false
	}
	// assume if we're asking that the context is present on the PR
	if sets.NewString(cp.RequiredIfPresentContexts...).Has(c) {
		return false
	}
	if cp.SkipUnknownContexts != nil && *cp.SkipUnknownContexts {
		return true
	}
	return false
}

// MissingRequiredContexts discard the optional contexts and only look of extra required contexts that are not provided.
func (cp *ContextPolicy) MissingRequiredContexts(contexts []string) []string {
	if len(cp.RequiredContexts) == 0 {
		return nil
	}
	existingContexts := sets.NewString()
	for _, c := range contexts {
		existingContexts.Insert(c)
	}
	var missingContexts []string
	for c := range sets.NewString(cp.RequiredContexts...).Difference(existingContexts) {
		missingContexts = append(missingContexts, c)
	}
	return missingContexts
}
