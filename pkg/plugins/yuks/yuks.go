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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/sirupsen/logrus"

	"github.com/jenkins-x/lighthouse/pkg/pluginhelp"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
)

var (
	match  = regexp.MustCompile(`(?mi)^/(?:lh-)?joke\s*$`)
	simple = regexp.MustCompile(`^[\w?'!., ]+$`)
)

const (
	jokeURL    = realJoke("https://icanhazdadjoke.com")
	pluginName = "yuks"
)

func createPlugin(j joker) plugins.Plugin {
	return plugins.Plugin{
		Description: "The yuks plugin comments with jokes in response to the `/joke` command.",
		Commands: []plugins.Command{{
			Filter: func(e scmprovider.GenericCommentEvent) bool { return e.Action == scm.ActionCreate },
			Regex:  match,
			GenericCommentHandler: func(_ []string, pc plugins.Agent, e scmprovider.GenericCommentEvent) error {
				return joke(pc.SCMProviderClient, pc.Logger, &e, j)
			},
			Help: []pluginhelp.Command{{
				Usage:       "/joke",
				Description: "Tells a joke.",
				Featured:    false,
				WhoCanUse:   "Anyone can use the `/joke` command.",
				Examples:    []string{"/joke", "/lh-joke"},
			}},
		}},
	}
}

func init() {
	plugins.RegisterPlugin(pluginName, createPlugin(jokeURL))
}

type scmProviderClient interface {
	CreateComment(owner, repo string, number int, pr bool, comment string) error
	QuoteAuthorForComment(string) string
}

type joker interface {
	readJoke() (string, error)
}

type realJoke string

var client = http.Client{}

type jokeResult struct {
	Joke string `json:"joke"`
}

func (url realJoke) readJoke() (string, error) {
	req, err := http.NewRequest("GET", string(url), nil)
	if err != nil {
		return "", fmt.Errorf("could not create request %s: %v", url, err)
	}
	req.Header.Add("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not read joke from %s: %v", url, err)
	}
	defer resp.Body.Close()
	var a jokeResult
	if err = json.NewDecoder(resp.Body).Decode(&a); err != nil {
		return "", err
	}
	if a.Joke == "" {
		return "", fmt.Errorf("result from %s did not contain a joke", url)
	}
	return a.Joke, nil
}

func joke(spc scmProviderClient, log *logrus.Entry, e *scmprovider.GenericCommentEvent, j joker) error {
	for i := 0; i < 10; i++ {
		resp, err := j.readJoke()
		if err != nil {
			return err
		}
		if simple.MatchString(resp) {
			log.Infof("Commenting with \"%s\".", resp)
			return spc.CreateComment(e.Repo.Namespace, e.Repo.Name, e.Number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, spc.QuoteAuthorForComment(e.Author.Login), resp))
		}

		log.Errorf("joke contains invalid characters: %v", resp)
	}

	return errors.New("all 10 jokes contain invalid character... such an unlucky day")
}
