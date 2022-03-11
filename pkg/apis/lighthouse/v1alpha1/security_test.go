package v1alpha1

import (
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

func TestLighthousePipelineSecurityPolicy_GetMaximumDurationForPipeline(t *testing.T) {
	policy := LighthousePipelineSecurityPolicy{}
	policy.Spec.Enforce.MaximumPipelineDuration = &metav1.Duration{Duration: time.Minute * 15}

	assert.Equal(t, "15m0s", policy.GetMaximumDurationForPipeline(&metav1.Duration{Duration: time.Hour}).Duration.String(), "job's timeout is higher than allowed timeout defined in policy, enforcing")
	assert.Equal(t, "9m0s", policy.GetMaximumDurationForPipeline(&metav1.Duration{Duration: time.Minute * 9}).Duration.String(), "job's timeout is lower, should not enforce from policy")
}

func TestLighthousePipelineSecurityPolicy_GetMaximumDurationForPipeline_DoesNotEnforceAnything(t *testing.T) {
	policy := LighthousePipelineSecurityPolicy{}
	policy.Spec.Enforce.MaximumPipelineDuration = &metav1.Duration{}

	assert.Equal(t, "24h0m0s", policy.GetMaximumDurationForPipeline(&metav1.Duration{Duration: time.Hour * 24}).Duration.String(), "nothing is to be enforced")
}

func TestLighthousePipelineSecurityPolicy_NothingIsDefinedThenNothingIsEnforced(t *testing.T) {
	policy := LighthousePipelineSecurityPolicy{}

	assert.False(t, policy.IsEnforcingMaximumPipelineDuration())
	assert.False(t, policy.IsEnforcingServiceAccount())
	assert.False(t, policy.IsEnforcingNamespace())
}

func TestLighthousePipelineSecurityPolicy_DefinedServiceAccountEnablesEnforcement(t *testing.T) {
	policy := LighthousePipelineSecurityPolicy{}
	policy.Spec.Enforce.ServiceAccountName = "something"

	assert.False(t, policy.IsEnforcingMaximumPipelineDuration())
	assert.True(t, policy.IsEnforcingServiceAccount())
	assert.False(t, policy.IsEnforcingNamespace())
}

func TestLighthousePipelineSecurityPolicy_DefinedNamespaceEnablesEnforcement(t *testing.T) {
	policy := LighthousePipelineSecurityPolicy{}
	policy.Spec.Enforce.Namespace = "jx-test2"

	assert.False(t, policy.IsEnforcingMaximumPipelineDuration())
	assert.False(t, policy.IsEnforcingServiceAccount())
	assert.True(t, policy.IsEnforcingNamespace())
}
