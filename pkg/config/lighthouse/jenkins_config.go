package lighthouse

import (
	"fmt"

	"k8s.io/apimachinery/pkg/labels"
)

// JenkinsConfig is config for the Jenkins controller.
type JenkinsConfig struct {
	Controller `json:",inline"`
	// LabelSelectorString compiles into LabelSelector at load time.
	// If set, this option needs to match --label-selector used by
	// the desired jenkins-operator. This option is considered
	// invalid when provided with a single jenkins-operator config.
	//
	// For label selector syntax, see below:
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors
	LabelSelectorString string `json:"label_selector,omitempty"`
	// LabelSelector is used so different jenkins-operator replicas
	// can use their own configuration.
	LabelSelector labels.Selector `json:"-"`
}

// Parse initializes and validates the Config
func (c *JenkinsConfig) Parse() error {
	if err := c.Controller.Parse(); err != nil {
		return fmt.Errorf("validating jenkins_operators config: %v", err)
	}
	sel, err := labels.Parse(c.LabelSelectorString)
	if err != nil {
		return fmt.Errorf("invalid jenkins_operators.label_selector option: %v", err)
	}
	c.LabelSelector = sel
	return nil
}
