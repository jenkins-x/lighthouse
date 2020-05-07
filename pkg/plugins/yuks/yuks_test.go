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

package yuks

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/driver/fake"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/sirupsen/logrus"
)

type fakeJoke string

var human = flag.Bool("human", false, "Enable to run additional manual tests")

func (j fakeJoke) readJoke() (string, error) {
	return string(j), nil
}

func TestRealJoke(t *testing.T) {
	if !*human {
		t.Skip("Real jokes disabled for automation. Manual users can add --human")
	}
	if joke, err := jokeURL.readJoke(); err != nil {
		t.Errorf("Could not read joke from %s: %v", jokeURL, err)
	} else {
		fmt.Println(joke)
	}
}

// Medium integration test (depends on ability to open a TCP port)
func TestJokesMedium(t *testing.T) {
	j := "What do you get when you cross a joke with a rhetorical question?"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"joke": "%s"}`, j)
	}))
	defer ts.Close()
	fakeScmClient, fc := fake.NewDefault()
	fakeClient := scmprovider.ToTestClient(fakeScmClient)

	comment := "/joke"

	e := &scmprovider.GenericCommentEvent{
		Action:     scm.ActionCreate,
		Body:       comment,
		Number:     5,
		IssueState: "open",
	}
	if err := handle(fakeClient, logrus.WithField("plugin", pluginName), e, realJoke(ts.URL)); err != nil {
		t.Errorf("didn't expect error: %v", err)
		return
	}
	if len(fc.IssueComments[5]) != 1 {
		t.Error("should have commented.")
		return
	}
	if c := fc.IssueComments[5][0]; !strings.Contains(c.Body, j) {
		t.Errorf("missing joke: %s from comment: %v", j, c)
	}
}

// Small, unit tests
func TestJokes(t *testing.T) {
	var testcases = []struct {
		name          string
		action        scm.Action
		body          string
		state         string
		joke          fakeJoke
		pr            bool
		shouldComment bool
		shouldError   bool
	}{
		{
			name:          "ignore edited comment",
			state:         "open",
			action:        scm.ActionUpdate,
			body:          "/joke",
			joke:          "this? that.",
			shouldComment: false,
			shouldError:   false,
		},
		{
			name:          "leave joke on pr",
			state:         "open",
			action:        scm.ActionCreate,
			body:          "/joke",
			joke:          "this? that.",
			pr:            true,
			shouldComment: true,
			shouldError:   false,
		},
		{
			name:          "leave joke on issue",
			state:         "open",
			action:        scm.ActionCreate,
			body:          "/joke",
			joke:          "this? that.",
			shouldComment: true,
			shouldError:   false,
		},
		{
			name:          "leave joke on issue, trailing space",
			state:         "open",
			action:        scm.ActionCreate,
			body:          "/joke \r",
			joke:          "this? that.",
			shouldComment: true,
			shouldError:   false,
		},
		{
			name:          "reject bad joke chars",
			state:         "open",
			action:        scm.ActionCreate,
			body:          "/joke",
			joke:          "[hello](url)",
			shouldComment: false,
			shouldError:   true,
		},
		{
			name:          "leave joke on pr with prefix",
			state:         "open",
			action:        scm.ActionCreate,
			body:          "/lh-joke",
			joke:          "this? that.",
			pr:            true,
			shouldComment: true,
			shouldError:   false,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			fakeScmClient, fc := fake.NewDefault()
			fakeClient := scmprovider.ToTestClient(fakeScmClient)

			e := &scmprovider.GenericCommentEvent{
				Action:     tc.action,
				Body:       tc.body,
				Number:     5,
				IssueState: tc.state,
				IsPR:       tc.pr,
			}
			err := handle(fakeClient, logrus.WithField("plugin", pluginName), e, tc.joke)
			if !tc.shouldError && err != nil {
				t.Fatalf("For case %s, didn't expect error: %v", tc.name, err)
			} else if tc.shouldError && err == nil {
				t.Fatalf("For case %s, expected an error to occur", tc.name)
			}
			var comments map[int][]*scm.Comment
			if tc.pr {
				comments = fc.PullRequestComments
			} else {
				comments = fc.IssueComments
			}
			if tc.shouldComment && len(comments[5]) != 1 {
				t.Errorf("For case %s, should have commented.", tc.name)
			} else if !tc.shouldComment && len(comments[5]) != 0 {
				t.Errorf("For case %s, should not have commented.", tc.name)
			}
		})
	}
}
