package security

import (
	"context"

	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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

// getPolicyNameAttachedToTektonPipelineRun retrieves a policy name from a label
func getPolicyNameAttachedToTektonPipelineRun(run *tektonv1beta1.PipelineRun) string {
	labels := run.GetLabels()
	if val, ok := labels[PolicyAnnotationName]; ok {
		return val
	}
	return ""
}
