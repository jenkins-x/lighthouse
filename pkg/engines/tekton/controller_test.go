package tekton

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	lighthousev1alpha1 "github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/stretchr/testify/assert"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

const (
	dashboardBaseURL = "https://example.com/"
)

type seededRandIDGenerator struct{}

func (s *seededRandIDGenerator) GenerateBuildID() string {
	return strconv.Itoa(utilrand.Int())
}
func TestReconcile(t *testing.T) {
	testCases := []string{
		"start-pullrequest",
		"update-job",
		"start-batch-pullrequest",
		"start-push",
	}

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			utilrand.Seed(12345)

			testData := path.Join("test_data", "controller", tc)
			_, err := os.Stat(testData)
			assert.NoError(t, err)

			// load observed state
			ns := "jx"
			observedPR, err := loadControllerPipelineRun(true, testData)
			assert.NoError(t, err)
			observedJob, err := loadLighthouseJob(true, testData)
			assert.NoError(t, err)
			observedPipeline, err := loadObservedPipeline(testData)
			assert.NoError(t, err)
			var state []runtime.Object
			if observedPR != nil {
				state = append(state, observedPR)
			}
			if observedJob != nil {
				state = append(state, observedJob)
			}
			if observedPipeline != nil {
				state = append(state, observedPipeline)
			}

			// load expected state
			expectedPR, err := loadControllerPipelineRun(false, testData)
			assert.NoError(t, err)
			expectedJob, err := loadLighthouseJob(false, testData)
			assert.NoError(t, err)

			// create fake controller
			scheme := runtime.NewScheme()
			err = lighthousev1alpha1.AddToScheme(scheme)
			assert.NoError(t, err)
			err = pipelinev1beta1.AddToScheme(scheme)
			assert.NoError(t, err)
			c := fake.NewFakeClientWithScheme(scheme, state...)
			reconciler := NewLighthouseJobReconciler(c, c, scheme, dashboardBaseURL, ns)
			reconciler.idGenerator = &seededRandIDGenerator{}

			// invoke reconcile
			_, err = reconciler.Reconcile(ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: ns,
					Name:      observedJob.GetName(),
				},
			})
			assert.NoError(t, err)

			// assert observed state matches expected state
			if expectedPR != nil {
				var pipelineRunList tektonv1beta1.PipelineRunList
				err := c.List(nil, &pipelineRunList, client.InNamespace(ns))
				assert.NoError(t, err)
				assert.Len(t, pipelineRunList.Items, 1)
				updatedPR := pipelineRunList.Items[0].DeepCopy()
				if d := cmp.Diff(expectedPR, updatedPR); d != "" {
					t.Errorf("PipelineRun did not match expected: %s", d)
					py, _ := yaml.Marshal(updatedPR)
					t.Logf("pr:\n%s", string(py))
				}
			}
			if expectedJob != nil {
				var jobList lighthousev1alpha1.LighthouseJobList
				err := c.List(nil, &jobList, client.InNamespace(ns))
				assert.NoError(t, err)
				assert.Len(t, jobList.Items, 1)
				// Ignore status.starttime since that's always going to be different
				updatedJob := jobList.Items[0].DeepCopy()
				updatedJob.Status.StartTime = metav1.Time{}
				if d := cmp.Diff(expectedJob, updatedJob); d != "" {
					t.Errorf("LighthouseJob did not match expected: %s", d)
				}
			}
		})
	}
}

func loadLighthouseJob(isObserved bool, dir string) (*v1alpha1.LighthouseJob, error) {
	var baseFn string
	if isObserved {
		baseFn = "observed-lhjob.yml"
	} else {
		baseFn = "expected-lhjob.yml"
	}
	fileName := filepath.Join(dir, baseFn)
	exists, err := util.FileExists(fileName)
	if err != nil {
		return nil, err
	}
	if exists {
		lhjob := &v1alpha1.LighthouseJob{}
		data, err := ioutil.ReadFile(fileName)
		if err != nil {
			return nil, err
		}
		err = yaml.Unmarshal(data, lhjob)
		if err != nil {
			return nil, err
		}
		return lhjob, err
	}
	return nil, nil
}

func loadControllerPipelineRun(isObserved bool, dir string) (*tektonv1beta1.PipelineRun, error) {
	var baseFn string
	if isObserved {
		baseFn = "observed-pr.yml"
	} else {
		baseFn = "expected-pr.yml"
	}
	fileName := filepath.Join(dir, baseFn)
	exists, err := util.FileExists(fileName)
	if err != nil {
		return nil, err
	}
	if exists {
		pr := &tektonv1beta1.PipelineRun{}
		data, err := ioutil.ReadFile(fileName)
		if err != nil {
			return nil, err
		}
		err = yaml.Unmarshal(data, pr)
		if err != nil {
			return nil, err
		}
		return pr, err
	}
	return nil, nil
}

func loadObservedPipeline(dir string) (*tektonv1beta1.Pipeline, error) {
	fileName := filepath.Join(dir, "observed-pipeline.yml")
	exists, err := util.FileExists(fileName)
	if err != nil {
		return nil, err
	}
	if exists {
		p := &tektonv1beta1.Pipeline{}
		data, err := ioutil.ReadFile(fileName)
		if err != nil {
			return nil, err
		}
		err = yaml.Unmarshal(data, p)
		if err != nil {
			return nil, err
		}
		return p, err
	}
	return nil, nil
}
