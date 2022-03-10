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
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

// ApplySecurityPolicyForTektonPipelineRun optionally applies enforcements from `kind: LighthousePipelineSecurityPolicy` if there was matched any during `kind: LighthouseJob` processing
func ApplySecurityPolicyForTektonPipelineRun(ctx context.Context, c client.Client, run *tektonv1beta1.PipelineRun, policiesNamespace string) error {
	// Tekton PipelineRun inherits labels from LighthouseJob, including a label that contains policy name
	policyName := getPolicyNameAttachedToTektonPipelineRun(run)

	// policies are optional
	if policyName == "" {
		logrus.Infof("No LighthousePipelineSecurityPolicy matched for this PipelineRun")
		return nil
	}

	var policy v1alpha1.LighthousePipelineSecurityPolicy
	// policy was assigned by web hooks handler, but is no longer accessible
	if err := c.Get(ctx, types.NamespacedName{Name: policyName, Namespace: policiesNamespace}, &policy); err != nil {
		return errors.Wrapf(err, "Cannot find LighthousePipelineSecurityPolicy of name %v in '%v' namespace", policyName, policiesNamespace)
	}

	// optionally apply service account name
	if policy.IsEnforcingServiceAccount() {
		logrus.Infof("Enforcing a serviceAccountName = %v", policy.Spec.Enforce.ServiceAccountName)
		run.Spec.ServiceAccountName = policy.Spec.Enforce.ServiceAccountName
	}

	// optionally set allowed maximum execution time
	if policy.IsEnforcingMaximumPipelineDuration() {
		//  enforces a maximum execution time to a pipeline in two cases:
		//      a) when it is not set explicitly
		//      b) when it is longer than maximum specified in the LighthousePipelineSecurityPolicy
		run.Spec.Timeout = policy.GetMaximumDurationForPipeline(run.Spec.Timeout)
	}

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

// getPolicyNameAttachedToTektonPipelineRun retrieves a policy name from a label
func getPolicyNameAttachedToTektonPipelineRun(run *tektonv1beta1.PipelineRun) string {
	labels := run.GetLabels()
	if val, ok := labels[PolicyAnnotationName]; ok {
		return val
	}
	return ""
}

// associateLighthouseJobWithPolicy marks a job that it hits a security policy. Later lighthouse-tekton-controller is using this link.
func associateLighthouseJobWithPolicy(request *v1alpha1.LighthouseJob, policy *v1alpha1.LighthousePipelineSecurityPolicy) {
	labels := request.GetLabels()
	labels[PolicyAnnotationName] = policy.Name
	request.SetLabels(labels)
}
