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

// Package pony adds pony images to the issue or PR in response to a /pony comment
package pony

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/sirupsen/logrus"
)

// Only the properties we actually use.
type ponyResult struct {
	Pony ponyResultPony `json:"pony"`
}

type ponyResultPony struct {
	Representations ponyRepresentations `json:"representations"`
}

type ponyRepresentations struct {
	Full  string `json:"full"`
	Small string `json:"small"`
}

const (
	ponyURL    = realHerd("https://theponyapi.com/api/v1/pony/random")
	pluginName = "pony"
)

func createPlugin(h herd) plugins.Plugin {
	return plugins.Plugin{
		Description: "The pony plugin adds a pony image to an issue or PR in response to the `/pony` command.",
		Commands: []plugins.Command{{
			Name: "pony",
			Arg: &plugins.CommandArg{
				Pattern:  ".+",
				Optional: true,
			},
			Description: "Add a little pony image to the issue or PR. A particular pony can optionally be named for a picture of that specific pony.",
			Action: plugins.
				Invoke(func(match plugins.CommandMatch, pc plugins.Agent, e scmprovider.GenericCommentEvent) error {
					return handle(match.Arg, pc.SCMProviderClient, pc.Logger, &e, h)
				}).
				When(plugins.Action(scm.ActionCreate)),
		}},
	}
}

func init() {
	plugins.RegisterPlugin(pluginName, createPlugin(ponyURL))
}

var client = http.Client{}

type scmProviderClient interface {
	CreateComment(owner, repo string, number int, pr bool, comment string) error
	QuoteAuthorForComment(string) string
}

type herd interface {
	readPony(string) (string, error)
}

type realHerd string

func formatURLs(small, full string) string {
	return fmt.Sprintf("[![pony image](%s)](%s)", small, full)
}

func (h realHerd) readPony(tags string) (string, error) {
	uri := string(h) + "?q=" + url.QueryEscape(tags)
	resp, err := client.Get(uri)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("no pony found")
	}
	var a ponyResult
	if err = json.NewDecoder(resp.Body).Decode(&a); err != nil {
		return "", fmt.Errorf("failed to decode response: %v", err)
	}

	embedded := a.Pony.Representations.Small
	tooBig, err := scmprovider.ImageTooBig(embedded)
	if err != nil {
		return "", fmt.Errorf("couldn't fetch pony for size check: %v", err)
	}
	if tooBig {
		return "", fmt.Errorf("the pony is too big")
	}
	return formatURLs(a.Pony.Representations.Small, a.Pony.Representations.Full), nil
}

func handle(tag string, spc scmProviderClient, log *logrus.Entry, e *scmprovider.GenericCommentEvent, p herd) error {
	for i := 0; i < 5; i++ {
		resp, err := p.readPony(tag)
		if err != nil {
			log.WithError(err).Println("Failed to get a pony")
			continue
		}
		return spc.CreateComment(e.Repo.Namespace, e.Repo.Name, e.Number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, spc.QuoteAuthorForComment(e.Author.Login), resp))
	}

	var msg string
	if tag != "" {
		msg = "Couldn't find a pony matching that query."
	} else {
		msg = "https://theponyapi.com appears to be down"
	}
	if err := spc.CreateComment(e.Repo.Namespace, e.Repo.Name, e.Number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, spc.QuoteAuthorForComment(e.Author.Login), msg)); err != nil {
		log.WithError(err).Error("Failed to leave comment")
	}

	return errors.New("could not find a valid pony image")
}
