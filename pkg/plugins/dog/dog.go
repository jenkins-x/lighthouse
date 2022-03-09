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

	"github.com/jenkins-x/lighthouse/pkg/plugins"
)

var (
	filetypes = regexp.MustCompile(`(?i)\.(jpg|gif|png)$`)
)

const (
	dogURL        = realPack("https://random.dog/woof.json")
	fineURL       = "https://storage.googleapis.com/this-is-fine-images/this_is_fine.png"
	notFineURL    = "https://storage.googleapis.com/this-is-fine-images/this_is_not_fine.png"
	unbearableURL = "https://storage.googleapis.com/this-is-fine-images/this_is_unbearable.jpg"
	pluginName    = "dog"
)

func createPlugin(p pack) plugins.Plugin {
	return plugins.Plugin{
		Description: "The dog plugin adds a dog image to an issue or PR in response to the `/woof` command.",
		Commands: []plugins.Command{{
			Name:        "woof|bark",
			Description: "Add a dog image to the issue or PR",
			Action: plugins.
				Invoke(func(_ plugins.CommandMatch, pc plugins.Agent, e scmprovider.GenericCommentEvent) error {
					return handle(pc.SCMProviderClient, pc.Logger, &e, p)
				}).
				When(plugins.Action(scm.ActionCreate)),
		}, {
			Name:        "this-is-fine",
			Description: "Add a dog image to the issue or PR",
			Action: plugins.
				Invoke(func(_ plugins.CommandMatch, pc plugins.Agent, e scmprovider.GenericCommentEvent) error {
					return formatURLAndSendResponse(pc.SCMProviderClient, &e, fineURL)
				}).
				When(plugins.Action(scm.ActionCreate)),
		}, {
			Name:        "this-is-not-fine",
			Description: "Add a dog image to the issue or PR",
			Action: plugins.
				Invoke(func(_ plugins.CommandMatch, pc plugins.Agent, e scmprovider.GenericCommentEvent) error {
					return formatURLAndSendResponse(pc.SCMProviderClient, &e, notFineURL)
				}).
				When(plugins.Action(scm.ActionCreate)),
		}, {
			Name:        "this-is-unbearable",
			Description: "Add a dog image to the issue or PR",
			Action: plugins.
				Invoke(func(_ plugins.CommandMatch, pc plugins.Agent, e scmprovider.GenericCommentEvent) error {
					return formatURLAndSendResponse(pc.SCMProviderClient, &e, unbearableURL)
				}).
				When(plugins.Action(scm.ActionCreate)),
		}},
	}
}

func init() {
	plugins.RegisterPlugin(pluginName, createPlugin(dogURL))
}

type scmProviderClient interface {
	CreateComment(owner, repo string, number int, pr bool, comment string) error
	QuoteAuthorForComment(string) string
}

type pack interface {
	readDog() (string, error)
}

type realPack string

var client = http.Client{}

type dogResult struct {
	URL string `json:"url"`
}

func formatURL(dogURL string) (string, error) {
	if dogURL == "" {
		return "", errors.New("empty url")
	}
	src, err := url.ParseRequestURI(dogURL)
	if err != nil {
		return "", fmt.Errorf("invalid url %s: %v", dogURL, err)
	}
	return fmt.Sprintf("[![dog image](%s)](%s)", src, src), nil
}

func (u realPack) readDog() (string, error) {
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
	// GitHub doesn't support videos :(
	if !filetypes.MatchString(a.URL) {
		return "", errors.New("unsupported doggo :( unknown filetype: " + a.URL)
	}
	// checking size, GitHub doesn't support big images
	toobig, err := scmprovider.ImageTooBig(a.URL)
	if err != nil {
		return "", err
	} else if toobig {
		return "", errors.New("unsupported doggo :( size too big: " + a.URL)
	}
	return a.URL, nil
}

func handle(spc scmProviderClient, log *logrus.Entry, e *scmprovider.GenericCommentEvent, p pack) error {
	for i := 0; i < 5; i++ {
		resp, err := p.readDog()
		if err != nil {
			log.WithError(err).Println("Failed to get dog img")
			continue
		}
		return formatURLAndSendResponse(spc, e, resp)
	}
	return errors.New("could not find a valid dog image")
}

func formatURLAndSendResponse(spc scmProviderClient, e *scmprovider.GenericCommentEvent, url string) error {
	msg, err := formatURL(url)
	if err != nil {
		return err
	}
	return sendResponse(spc, e, msg)
}

func sendResponse(spc scmProviderClient, e *scmprovider.GenericCommentEvent, msg string) error {
	return spc.CreateComment(e.Repo.Namespace, e.Repo.Name, e.Number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, spc.QuoteAuthorForComment(e.Author.Login), msg))
}
