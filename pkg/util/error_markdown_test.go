package util_test

import (
	"testing"

	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

var fileLink = "https://github.com/cb-kubecd/bdd-gh-1602679032/blob/master/"

func TestErrorToMarkdown(t *testing.T) {
	err := errors.Errorf("failed to do something")
	actual := util.ErrorToMarkdown(err, fileLink)
	assert.Equal(t, "* failed to do something\n", actual, "for %s", err.Error())

	err = errors.Wrapf(err, "top level error")
	actual = util.ErrorToMarkdown(err, fileLink)
	assert.Equal(t, "* top level error\n* failed to do something\n", actual, "for %s", err.Error())
}

func TestErrorToMarkdownWithfileLink(t *testing.T) {
	err := errors.Errorf("failed to load file .lighthouse/lint/triggers.yaml")

	err = errors.Wrapf(err, "top level error")
	actual := util.ErrorToMarkdown(err, fileLink)
	assert.Equal(t, "* top level error\n* failed to load file [.lighthouse/lint/triggers.yaml](https://github.com/cb-kubecd/bdd-gh-1602679032/blob/master/.lighthouse/lint/triggers.yaml)\n", actual, "for %s", err.Error())
}
