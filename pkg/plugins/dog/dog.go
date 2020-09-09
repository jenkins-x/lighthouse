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

// Package dog adds dog images to the issue or PR in response to a /woof comment
package dog

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/sirupsen/logrus"

	"github.com/jenkins-x/lighthouse/pkg/pluginhelp"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
)

var (
	match           = regexp.MustCompile(`(?mi)^/(?:lh-)?(woof|bark)\s*$`)
	fineRegex       = regexp.MustCompile(`(?mi)^/(?:lh-)?this-is-fine\s*$`)
	notFineRegex    = regexp.MustCompile(`(?mi)^/(?:lh-)?this-is-not-fine\s*$`)
	unbearableRegex = regexp.MustCompile(`(?mi)^/(?:lh-)?this-is-unbearable\s*$`)
	filetypes       = regexp.MustCompile(`(?i)\.(jpg|gif|png)$`)
)

const (
	dogURL        = realPack("https://random.dog/woof.json")
	fineURL       = "https://storage.googleapis.com/this-is-fine-images/this_is_fine.png"
	notFineURL    = "https://storage.googleapis.com/this-is-fine-images/this_is_not_fine.png"
	unbearableURL = "https://storage.googleapis.com/this-is-fine-images/this_is_unbearable.jpg"
	pluginName    = "dog"
)

var (
	plugin = plugins.Plugin{
		Description: "The dog plugin adds a dog image to an issue or PR in response to the `/woof` command.",
		Commands: []plugins.Command{{
			GenericCommentHandler: handleGenericComment,
			Filter:                func(e scmprovider.GenericCommentEvent) bool { return e.Action == scm.ActionCreate },
			Help: []pluginhelp.Command{{
				Usage:       "/(lh-)?(woof|bark|this-is-{fine|not-fine|unbearable})",
				Description: "Add a dog image to the issue or PR",
				Featured:    false,
				WhoCanUse:   "Anyone",
				Examples:    []string{"/woof", "/bark", "this-is-{fine|not-fine|unbearable}"},
			}},
		}},
	}
)

func init() {
	plugins.RegisterPlugin(pluginName, plugin)
}

type scmProviderClient interface {
	CreateComment(owner, repo string, number int, pr bool, comment string) error
	QuoteAuthorForComment(string) string
}

type pack interface {
	readDog(dogURL string) (string, error)
}

type realPack string

var client = http.Client{}

type dogResult struct {
	URL string `json:"url"`
}

// FormatURL will return the GH markdown to show the image for a specific dogURL.
func FormatURL(dogURL string) (string, error) {
	if dogURL == "" {
		return "", errors.New("empty url")
	}
	src, err := url.ParseRequestURI(dogURL)
	if err != nil {
		return "", fmt.Errorf("invalid url %s: %v", dogURL, err)
	}
	return fmt.Sprintf("[![dog image](%s)](%s)", src, src), nil
}

func (u realPack) readDog(dogURL string) (string, error) {
	if dogURL == "" {
		uri := string(u)
		req, err := http.NewRequest("GET", uri, nil)
		if err != nil {
			return "", fmt.Errorf("could not create request %s: %v", uri, err)
		}
		req.Header.Add("Accept", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return "", fmt.Errorf("could not read dog from %s: %v", uri, err)
		}
		defer resp.Body.Close()
		var a dogResult
		if err = json.NewDecoder(resp.Body).Decode(&a); err != nil {
			return "", err
		}
		dogURL = a.URL
	}

	// GitHub doesn't support videos :(
	if !filetypes.MatchString(dogURL) {
		return "", errors.New("unsupported doggo :( unknown filetype: " + dogURL)
	}
	// checking size, GitHub doesn't support big images
	toobig, err := scmprovider.ImageTooBig(dogURL)
	if err != nil {
		return "", err
	} else if toobig {
		return "", errors.New("unsupported doggo :( size too big: " + dogURL)
	}
	return FormatURL(dogURL)
}

func handleGenericComment(match []string, pc plugins.Agent, e scmprovider.GenericCommentEvent) error {
	return handle(match, pc.SCMProviderClient, pc.Logger, &e, dogURL)
}

func handle(_ []string, spc scmProviderClient, log *logrus.Entry, e *scmprovider.GenericCommentEvent, p pack) error {
	// Make sure they are requesting a dog
	mat := match.FindStringSubmatch(e.Body)
	url := ""
	if mat == nil {
		// check is this one of the famous.dog
		if fineRegex.FindStringSubmatch(e.Body) != nil {
			url = fineURL
		} else if notFineRegex.FindStringSubmatch(e.Body) != nil {
			url = notFineURL
		} else if unbearableRegex.FindStringSubmatch(e.Body) != nil {
			url = unbearableURL
		}

		if url == "" {
			return nil
		}
	}

	org := e.Repo.Namespace
	repo := e.Repo.Name
	number := e.Number

	for i := 0; i < 5; i++ {
		resp, err := p.readDog(url)
		if err != nil {
			log.WithError(err).Println("Failed to get dog img")
			continue
		}
		return spc.CreateComment(org, repo, number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, spc.QuoteAuthorForComment(e.Author.Login), resp))
	}

	return errors.New("could not find a valid dog image")
}
