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

package dog

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/driver/fake"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/sirupsen/logrus"
)

type fakePack string

var human = flag.Bool("human", false, "Enable to run additional manual tests")

func (c fakePack) readDog() (string, error) {
	return string(c), nil
}

func TestRealDog(t *testing.T) {
	if !*human {
		t.Skip("Real dogs disabled for automation. Manual users can add --human [--category=foo]")
	}
	if dog, err := dogURL.readDog(); err != nil {
		t.Errorf("Could not read dog from %s: %v", dogURL, err)
	} else {
		fmt.Println(dog)
	}
}

func TestFormat(t *testing.T) {
	re := regexp.MustCompile(`\[!\[.+\]\(.+\)\]\(.+\)`)
	basicURL := "http://example.com"
	testcases := []struct {
		name string
		url  string
		err  bool
	}{
		{
			name: "basically works",
			url:  basicURL,
			err:  false,
		},
		{
			name: "empty url",
			url:  "",
			err:  true,
		},
		{
			name: "bad url",
			url:  "http://this is not a url",
			err:  true,
		},
	}
	for _, tc := range testcases {
		ret, err := formatURL(tc.url)
		switch {
		case tc.err:
			if err == nil {
				t.Errorf("%s: failed to raise an error", tc.name)
			}
		case err != nil:
			t.Errorf("%s: unexpected error: %v", tc.name, err)
		case !re.MatchString(ret):
			t.Errorf("%s: bad return value: %s", tc.name, ret)
		}
	}
}

// Medium integration test (depends on ability to open a TCP port)
func TestHttpResponse(t *testing.T) {
	// create test cases for handling content length of images
	contentLength := make(map[string]string)
	contentLength["/dog.jpg"] = "717987"
	contentLength["/doggo.mp4"] = "37943259"
	contentLength["/bigdog.jpg"] = "12647753"

	// fake server for images
	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s, ok := contentLength[r.URL.Path]; ok {
			body := "binary image"
			w.Header().Set("Content-Length", s)
			io.WriteString(w, body)
		} else {
			t.Errorf("Cannot find content length for %s", r.URL.Path)
		}
	}))
	defer ts2.Close()

	// setup a stock valid request
	url := ts2.URL + "/dog.jpg"
	b, err := json.Marshal(&dogResult{
		URL: url,
	})
	if err != nil {
		t.Errorf("Failed to encode test data: %v", err)
	}

	// create test cases for handling http responses
	validResponse := string(b)
	var testcases = []struct {
		name     string
		comment  string
		path     string
		response string
		expected string
		isValid  bool
	}{
		{
			name:     "valid",
			comment:  "/woof",
			path:     "/valid",
			response: validResponse,
			expected: url,
			isValid:  true,
		},
		{
			name:     "invalid JSON",
			comment:  "/woof",
			path:     "/bad-json",
			response: `{"bad-blob": "not-a-url"`,
			isValid:  false,
		},
		{
			name:     "invalid URL",
			comment:  "/woof",
			path:     "/bad-url",
			response: `{"url": "not a url.."}`,
			isValid:  false,
		},
		{
			name:     "mp4 doggo unsupported :(",
			comment:  "/woof",
			path:     "/mp4-doggo",
			response: fmt.Sprintf(`{"url": "%s/doggo.mp4"}`, ts2.URL),
			isValid:  false,
		},
		{
			name:     "image too big",
			comment:  "/woof",
			path:     "/too-big",
			response: fmt.Sprintf(`{"url": "%s/bigdog.jpg"}`, ts2.URL),
			isValid:  false,
		},
		{
			name:     "this is fine",
			comment:  "/this-is-fine",
			expected: "this_is_fine.png",
			isValid:  true,
		},
		{
			name:     "this is not fine",
			comment:  "/this-is-not-fine",
			expected: "this_is_not_fine.png",
			isValid:  true,
		},
		{
			name:     "this is unbearable",
			comment:  "/this-is-unbearable",
			expected: "this_is_unbearable.jpg",
			isValid:  true,
		},
	}

	// fake server for image urls
	pathToResponse := make(map[string]string)
	for _, testcase := range testcases {
		pathToResponse[testcase.path] = testcase.response
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r, ok := pathToResponse[r.URL.Path]; ok {
			io.WriteString(w, r)
		} else {
			io.WriteString(w, validResponse)
		}
	}))
	defer ts.Close()

	// run test for each case
	for _, testcase := range testcases {
		dog, err := realPack(ts.URL + testcase.path).readDog()
		if testcase.isValid && err != nil {
			t.Errorf("For case %s, didn't expect error: %v", testcase.name, err)
		} else if !testcase.isValid && err == nil {
			t.Errorf("For case %s, expected error, received dog: %s", testcase.name, dog)
		}

		if !testcase.isValid {
			continue
		}

		// github fake client
		fakeScmClient, fc := fake.NewDefault()
		fakeClient := scmprovider.ToTestClient(fakeScmClient)

		// fully test handling a comment
		e := &scmprovider.GenericCommentEvent{
			Action:     scm.ActionCreate,
			Body:       testcase.comment,
			Number:     5,
			IssueState: "open",
		}
		agent := plugins.Agent{
			SCMProviderClient: &fakeClient.Client,
			Logger:            logrus.WithField("plugin", pluginName),
		}
		plugin := createPlugin(realPack(ts.URL))
		err = plugin.InvokeCommandHandler(e, func(handler plugins.CommandEventHandler, e *scmprovider.GenericCommentEvent, match plugins.CommandMatch) error {
			return handler(match, agent, *e)
		})
		if err != nil {
			t.Errorf("tc %s: For comment %s, didn't expect error: %v", testcase.name, testcase.comment, err)
		}

		if len(fc.IssueComments[5]) != 1 {
			t.Errorf("tc %s: should have commented", testcase.name)
		}
		if c := fc.IssueComments[5][0]; !strings.Contains(c.Body, testcase.expected) {
			t.Errorf("tc %s: missing image url: %s from comment: %v", testcase.name, testcase.expected, c.Body)
		}
	}
}

// Small, unit tests
func TestDogs(t *testing.T) {
	var testcases = []struct {
		name          string
		action        scm.Action
		body          string
		state         string
		pr            bool
		shouldComment bool
	}{
		{
			name:          "ignore edited comment",
			state:         "open",
			action:        scm.ActionUpdate,
			body:          "/woof",
			shouldComment: false,
		},
		{
			name:          "leave dog on pr",
			state:         "open",
			action:        scm.ActionCreate,
			body:          "/woof",
			pr:            true,
			shouldComment: true,
		},
		{
			name:          "leave dog on pr with prefix",
			state:         "open",
			action:        scm.ActionCreate,
			body:          "/lh-woof",
			pr:            true,
			shouldComment: true,
		},
		{
			name:          "leave dog on issue",
			state:         "open",
			action:        scm.ActionCreate,
			body:          "/woof",
			shouldComment: true,
		},
		{
			name:          "leave dog on issue, trailing space",
			state:         "open",
			action:        scm.ActionCreate,
			body:          "/woof \r",
			shouldComment: true,
		},
		{
			name:          "leave dog on issue, trailing /bark",
			state:         "open",
			action:        scm.ActionCreate,
			body:          "/bark",
			shouldComment: true,
		},
		{
			name:          "leave dog on issue, trailing /bark, trailing space",
			state:         "open",
			action:        scm.ActionCreate,
			body:          "/bark \r",
			shouldComment: true,
		},
		{
			name:          "leave this-is-fine on pr",
			state:         "open",
			action:        scm.ActionCreate,
			body:          "/this-is-fine",
			pr:            true,
			shouldComment: true,
		},
		{
			name:          "leave this-is-fine on pr with prefix",
			state:         "open",
			action:        scm.ActionCreate,
			body:          "/lh-this-is-fine",
			pr:            true,
			shouldComment: true,
		},
		{
			name:          "leave this-is-not-fine on pr",
			state:         "open",
			action:        scm.ActionCreate,
			body:          "/this-is-not-fine",
			pr:            true,
			shouldComment: true,
		},
		{
			name:          "leave this-is-unbearable on pr",
			state:         "open",
			action:        scm.ActionCreate,
			body:          "/this-is-unbearable",
			pr:            true,
			shouldComment: true,
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
			agent := plugins.Agent{
				SCMProviderClient: &fakeClient.Client,
				Logger:            logrus.WithField("plugin", pluginName),
			}
			plugin := createPlugin(fakePack("http://127.0.0.1"))
			err := plugin.InvokeCommandHandler(e, func(handler plugins.CommandEventHandler, e *scmprovider.GenericCommentEvent, match plugins.CommandMatch) error {
				return handler(match, agent, *e)
			})
			if err != nil {
				t.Errorf("For case %s, didn't expect error: %v", tc.name, err)
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
