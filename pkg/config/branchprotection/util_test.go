/*
Copyright 2018 The Kubernetes Authors.

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

package branchprotection

import (
	"reflect"
	"sort"
	"testing"
)

var (
	y   = true
	n   = false
	yes = &y
	no  = &n
)

func TestSelectBool(t *testing.T) {
	cases := []struct {
		name     string
		parent   *bool
		child    *bool
		expected *bool
	}{
		{
			name: "default is nil",
		},
		{
			name:     "use child if set",
			child:    yes,
			expected: yes,
		},
		{
			name:     "child overrides parent",
			child:    yes,
			parent:   no,
			expected: yes,
		},
		{
			name:     "use parent if child unset",
			parent:   no,
			expected: no,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := selectBool(tc.parent, tc.child)
			if !reflect.DeepEqual(actual, tc.expected) {
				t.Errorf("actual %v != expected %v", actual, tc.expected)
			}
		})
	}
}

func TestSelectInt(t *testing.T) {
	one := 1
	two := 2
	cases := []struct {
		name     string
		parent   *int
		child    *int
		expected *int
	}{
		{
			name: "default is nil",
		},
		{
			name:     "use child if set",
			child:    &one,
			expected: &one,
		},
		{
			name:     "child overrides parent",
			child:    &one,
			parent:   &two,
			expected: &one,
		},
		{
			name:     "use parent if child unset",
			parent:   &two,
			expected: &two,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := selectInt(tc.parent, tc.child)
			if !reflect.DeepEqual(actual, tc.expected) {
				t.Errorf("actual %v != expected %v", actual, tc.expected)
			}
		})
	}
}

func TestUnionStrings(t *testing.T) {
	cases := []struct {
		name     string
		parent   []string
		child    []string
		expected []string
	}{
		{
			name: "empty list",
		},
		{
			name:     "all parent items",
			parent:   []string{"hi", "there"},
			expected: []string{"hi", "there"},
		},
		{
			name:     "all child items",
			child:    []string{"hi", "there"},
			expected: []string{"hi", "there"},
		},
		{
			name:     "both child and parent items, no duplicates",
			child:    []string{"hi", "world"},
			parent:   []string{"hi", "there"},
			expected: []string{"hi", "there", "world"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := unionStrings(tc.parent, tc.child)
			sort.Strings(actual)
			sort.Strings(tc.expected)
			if !reflect.DeepEqual(actual, tc.expected) {
				t.Errorf("actual %v != expected %v", actual, tc.expected)
			}
		})
	}
}
