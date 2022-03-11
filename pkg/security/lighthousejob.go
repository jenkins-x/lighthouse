package security

//
// Lighthouse Pipeline Security
// ============================
//   See ./README.md
//

import (
	"context"
	"fmt"

	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	clientset "github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ScmInfo interface {
	GetFullRepositoryName() string
}

// ApplySecurityPolicyForLighthouseJob applies security restrictions if ANY policy was matched
func ApplySecurityPolicyForLighthouseJob(c clientset.Interface, request *v1alpha1.LighthouseJob, repo ScmInfo, policiesNs string) error {
	policy, err, matchedAnyPolicy := findSecurityPolicyForRepository(c, repo.GetFullRepositoryName(), policiesNs)
	if err != nil {
		return errors.Wrapf(err, "cannot apply security policy for a job")
	}
	if matchedAnyPolicy {
		logrus.Infof("Selected LighthousePipelineSecurityPolicy name=%v for repository %v", policy.Name, repo.GetFullRepositoryName())

		// optionally enforce a namespace
		if policy.IsEnforcingNamespace() {
			request.SetNamespace(policy.Spec.Enforce.Namespace)
		}

		// mark a job that it hits a security policy
		associateLighthouseJobWithPolicy(request, &policy)

		return nil
	}
	logrus.Infof("Job '%s' does not match any LighthousePipelineSecurityPolicy, not applying any policy", request.Name)
	return nil
}

// findSecurityPolicyForRepository Finds a valid `kind: LighthousePipelineSecurityPolicy` that selector `.spec.RepositoryPattern` would match full repository name (do not confuse with url)
func findSecurityPolicyForRepository(c clientset.Interface, repository string, policiesNs string) (v1alpha1.LighthousePipelineSecurityPolicy, error, bool) {
	policies, err := c.LighthouseV1alpha1().LighthousePipelineSecurityPolicies(policiesNs).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		return v1alpha1.LighthousePipelineSecurityPolicy{}, errors.Wrapf(err, "API returned error while looking for LighthousePipelineSecurityPolicy"), false
	}
	var policy v1alpha1.LighthousePipelineSecurityPolicy
	matched := 0

	for _, currentPolicy := range policies.Items {
		match, err := currentPolicy.Spec.IsRepositoryMatchingPattern(repository)

		// this will block all jobs from running, as we have 'some policies' but they are useless. Invalid policies cannot be disabling security at all
		if err != nil {
			return v1alpha1.LighthousePipelineSecurityPolicy{}, errors.Wrapf(err, "panic! invalid repository pattern in LighthousePipelineSecurityPolicy of name %v", policy.Name), false
		}
		if match {
			logrus.Infof("Matched LighthousePipelineSecurityPolicy name=%v for repository %v", currentPolicy.Name, repository)
			matched += 1
			policy = currentPolicy
		}
	}

	// security configuration error: policies are incorrectly configured and multiple policies are matching single repository
	// this will block from scheduling a job from repository that matches multiple security policies
	if matched > 1 {
		logrus.Errorln("Too many policies matched")
		return v1alpha1.LighthousePipelineSecurityPolicy{}, errors.New(fmt.Sprintf("too many policies matched for repository %v", repository)), false
	}

	// no security policy matched, no restrictions to apply
	if matched == 0 {
		return v1alpha1.LighthousePipelineSecurityPolicy{}, nil, false
	}

	return policy, nil, true
}

// associateLighthouseJobWithPolicy marks a job that it hits a security policy. Later lighthouse-tekton-controller is using this link.
func associateLighthouseJobWithPolicy(request *v1alpha1.LighthouseJob, policy *v1alpha1.LighthousePipelineSecurityPolicy) {
	labels := request.GetLabels()
	labels[PolicyAnnotationName] = policy.Name
	request.SetLabels(labels)
}
