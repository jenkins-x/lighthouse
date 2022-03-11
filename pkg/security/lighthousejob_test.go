package security

import (
	"testing"

	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
)

func TestAssociateLighthouseJobWithPolicy(t *testing.T) {
	policy := v1alpha1.LighthousePipelineSecurityPolicy{}
	policy.Name = "mypolicy"

	job := v1alpha1.LighthouseJob{}
	job.Labels = map[string]string{}

	associateLighthouseJobWithPolicy(&job, &policy)

	label, _ := job.Labels["lighthouse.jenkins-x.io/securityPolicyName"]
	assert.Equal(t, "mypolicy", label)
}

// Given POLICY 'mypolicy' enforces a namespace 'jx-test1' for LighthouseJob spawned from repositories matching 'my-repo/symfony-([0-9]+)-app' regexp
// When we submit a LighthouseJob in 'jx' namespace from repository 'my-repo/symfony-5-app'
// Then a namespace for LighthouseJob will be set to 'jx-test1' because it matches the regexp pattern
func TestApplySecurityPolicyForLighthouseJob_EnforcesNamespace(t *testing.T) {
	policy := v1alpha1.LighthousePipelineSecurityPolicy{}
	policy.Name = "mypolicy"
	policy.Namespace = "jx"
	policy.Spec.RepositoryPattern = "my-repo/symfony-([0-9]+)-app"
	policy.Spec.Enforce.Namespace = "jx-test1"

	job := v1alpha1.LighthouseJob{}
	job.Name = "pr-symfony-5-app-1"
	job.Namespace = "jx"
	job.Labels = map[string]string{}

	c := fake.NewSimpleClientset(&policy)
	err := ApplySecurityPolicyForLighthouseJob(c, &job, &testRepoInformation{"my-repo/symfony-5-app"}, "jx")

	assert.Nil(t, err)
	assert.Equal(t, "jx-test1", job.Namespace, "expected that ApplySecurityPolicyForLighthouseJob() will mutate the Namespace field of a LighthouseJob")
}

// Given there are no policies defined at all
// Then nothing will be applied for a LighthouseJob
func TestApplySecurityPolicyForLighthouseJob_DoesNothingAsNoPoliciesAreDefined(t *testing.T) {
	job := v1alpha1.LighthouseJob{}
	job.Name = "release-gin-app"
	job.Namespace = "jx"
	job.Labels = map[string]string{}

	c := fake.NewSimpleClientset() // no policies added to the mock, no policies available
	err := ApplySecurityPolicyForLighthouseJob(c, &job, &testRepoInformation{"my-repo/gin-app"}, "jx")

	assert.Nil(t, err, "expected that when no policies are defined, then jobs will be scheduled without interruption")
	assert.Equal(t, "jx", job.Namespace, "expected that namespace filed would be not touched, as there were no policy to apply")
}

// Given there is a defined EMPTY POLICY
// And that policy will be MATCHED by a LighthouseJob
// Then NOTHING happens, as the policy is empty
func TestApplySecurityPolicyForLighthouseJob_DoesNothingAsMatchedPolicyIsEmpty(t *testing.T) {
	policy := v1alpha1.LighthousePipelineSecurityPolicy{}
	policy.Name = "mypolicy"
	policy.Namespace = "jx"
	policy.Spec.RepositoryPattern = "my-repo/symfony-([0-9]+)-app"

	job := v1alpha1.LighthouseJob{}
	job.Name = "release-gin-app"
	job.Namespace = "jx"
	job.Labels = map[string]string{}

	c := fake.NewSimpleClientset(&policy)
	err := ApplySecurityPolicyForLighthouseJob(c, &job, &testRepoInformation{"my-repo/symfony-161-app"}, "jx")

	assert.Nil(t, err, "expected that empty policy would not block from scheduling a pipeline")
	assert.Equal(t, "jx", job.Namespace, "expected that the namespace would not be touched")
}

// Given we have defined at least one policy which repository matching pattern is not a valid regexp
// Then no any job could be scheduled, as we have unclear situation what could be safe to schedule
func TestApplySecurityPolicyForLighthouseJob_BrokenPolicyPreventsSchedulingJobs(t *testing.T) {
	policy := v1alpha1.LighthousePipelineSecurityPolicy{}
	policy.Name = "mypolicy"
	policy.Namespace = "jx"
	policy.Spec.RepositoryPattern = "/\\/in/v\\a\\llid"

	c := fake.NewSimpleClientset(&policy)
	err := ApplySecurityPolicyForLighthouseJob(c, &v1alpha1.LighthouseJob{}, &testRepoInformation{"some-repo"}, "jx")

	assert.Contains(t, err.Error(), "cannot apply security policy for a job: panic! invalid repository pattern in LighthousePipelineSecurityPolicy")
	assert.Contains(t, err.Error(), "cannot compile regexp of `kind: LighthousePipelineSecurityPolicy")
}

// Given there are MORE policies matching the same repository
// Then fail a job that is to be scheduled from that repository
func TestApplySecurityPolicyForLighthouseJob_TooManyMatchedPoliciesWillPreventThatJobFromScheduling(t *testing.T) {
	policy1 := v1alpha1.LighthousePipelineSecurityPolicy{}
	policy1.Name = "mypolicy-1"
	policy1.Namespace = "jx"
	policy1.Spec.RepositoryPattern = "my-repo"

	policy2 := v1alpha1.LighthousePipelineSecurityPolicy{}
	policy2.Name = "mypolicy-2"
	policy2.Namespace = "jx"
	policy2.Spec.RepositoryPattern = "my-repo"

	c := fake.NewSimpleClientset(&policy1, &policy2)

	// matched 2 policies by "my-repo" pattern: will fail
	err := ApplySecurityPolicyForLighthouseJob(c, &v1alpha1.LighthouseJob{}, &testRepoInformation{"my-repo"}, "jx")
	assert.Equal(t, "cannot apply security policy for a job: too many policies matched for repository my-repo", err.Error())

	// not matched any policy, will not fail
	result := ApplySecurityPolicyForLighthouseJob(c, &v1alpha1.LighthouseJob{}, &testRepoInformation{"no-matching-repo"}, "jx")
	assert.Nil(t, result)
}

type testRepoInformation struct {
	MockedFullRepositoryName string
}

func (r *testRepoInformation) GetFullRepositoryName() string {
	return r.MockedFullRepositoryName
}
