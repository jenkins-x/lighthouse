package tekton

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/jenkins-x/lighthouse/pkg/watcher"

	"github.com/stretchr/testify/require"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	lighthousev1alpha1 "github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	fakelh "github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned/fake"

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
	dashboardBaseURL  = "https://example.com/"
	dashboardTemplate = "#/namespaces/{{ .Namespace }}/pipelineruns/{{ .PipelineRun }}"
)

var (
	// generateTestOutput enable to regenerate the expected output
	generateTestOutput = false
)

type seededRandIDGenerator struct{}

func (s *seededRandIDGenerator) GenerateBuildID() string {
	return strconv.Itoa(utilrand.Int())
}

func TestReconcile(t *testing.T) {
	testCases := []string{
		"debug-pr-no-taskRunSpecs",
		"debug-pr",
		"update-job",
		"start-pullrequest",
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
			observedPR, _, err := loadControllerPipelineRun(true, testData)
			assert.NoError(t, err)
			observedJob, _, err := loadLighthouseJob(true, testData)
			assert.NoError(t, err)
			observedPipeline, err := loadObservedPipeline(testData)
			assert.NoError(t, err)
			var state []client.Object
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
			expectedPR, expectedPRFile, err := loadControllerPipelineRun(false, testData)
			assert.NoError(t, err)
			expectedJob, expectedJobFile, err := loadLighthouseJob(false, testData)
			assert.NoError(t, err)

			// create fake controller
			scheme := runtime.NewScheme()
			err = lighthousev1alpha1.AddToScheme(scheme)
			assert.NoError(t, err)
			err = pipelinev1beta1.AddToScheme(scheme)
			assert.NoError(t, err)

			lhClient := fakelh.NewSimpleClientset()

			if strings.HasPrefix(tc, "debug") {
				branch := "master"
				if tc == "debug-pr-no-taskRunSpecs" {
					branch = "PR-813"
				}
				lhClient = fakelh.NewSimpleClientset(
					&lighthousev1alpha1.LighthouseBreakpoint{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-bp",
							Namespace: ns,
						},
						Spec: lighthousev1alpha1.LighthouseBreakpointSpec{
							Filter: lighthousev1alpha1.LighthousePipelineFilter{
								Owner:      "jenkins-x",
								Repository: "lighthouse",
								Branch:     branch,
								Context:    "github",
								Task:       "",
							},
							Debug: tektonv1beta1.TaskRunDebug{
								Breakpoint: []string{"onFailure"},
							},
						},
					},
				)
			}
			bpWatcher, err := watcher.NewBreakpointWatcher(lhClient, ns, nil)
			require.NoError(t, err, "failed to create BreakpointWatcher")
			defer bpWatcher.Stop()

			c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(state...).Build()
			reconciler := NewLighthouseJobReconciler(c, c, scheme, dashboardBaseURL, dashboardTemplate, ns, bpWatcher.GetBreakpoints)
			reconciler.idGenerator = &seededRandIDGenerator{}
			reconciler.disableLogging = true

			// invoke reconcile
			_, err = reconciler.Reconcile(context.TODO(), ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: ns,
					Name:      observedJob.GetName(),
				},
			})
			assert.NoError(t, err)

			// assert observed state matches expected state
			if expectedPR != nil || generateTestOutput {
				var pipelineRunList tektonv1beta1.PipelineRunList
				err := c.List(nil, &pipelineRunList, client.InNamespace(ns))
				assert.NoError(t, err)
				assert.Len(t, pipelineRunList.Items, 1)
				updatedPR := pipelineRunList.Items[0].DeepCopy()
				if generateTestOutput {
					data, err := yaml.Marshal(updatedPR)
					require.NoError(t, err, "failed to marshal expected PR %#v", updatedPR)
					err = ioutil.WriteFile(expectedPRFile, data, 0644)
					require.NoError(t, err, "failed to save file %s", expectedPRFile)
					t.Logf("saved expected PR file %s\n", expectedPRFile)
				} else {
					if d := cmp.Diff(expectedPR, updatedPR); d != "" {
						t.Errorf("PipelineRun did not match expected: %s", d)
						py, _ := yaml.Marshal(updatedPR)
						t.Logf("pr:\n%s", string(py))
					}
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
				if generateTestOutput {
					data, err := yaml.Marshal(updatedJob)
					require.NoError(t, err, "failed to marshal expected job %#v", updatedJob)
					err = ioutil.WriteFile(expectedJobFile, data, 0644)
					require.NoError(t, err, "failed to save file %s", expectedJobFile)
					t.Logf("saved expected Job file %s\n", expectedJobFile)
				} else {
					if d := cmp.Diff(expectedJob, updatedJob); d != "" {
						t.Errorf("LighthouseJob did not match expected: %s", d)
					}
				}
			}
		})
	}
}

func loadLighthouseJob(isObserved bool, dir string) (*v1alpha1.LighthouseJob, string, error) {
	var baseFn string
	if isObserved {
		baseFn = "observed-lhjob.yml"
	} else {
		baseFn = "expected-lhjob.yml"
	}
	fileName := filepath.Join(dir, baseFn)
	exists, err := util.FileExists(fileName)
	if err != nil {
		return nil, fileName, err
	}
	if exists {
		lhjob := &v1alpha1.LighthouseJob{}
		data, err := ioutil.ReadFile(fileName)
		if err != nil {
			return nil, fileName, err
		}
		err = yaml.Unmarshal(data, lhjob)
		if err != nil {
			return nil, fileName, err
		}
		return lhjob, fileName, err
	}
	return nil, fileName, nil
}

func loadControllerPipelineRun(isObserved bool, dir string) (*tektonv1beta1.PipelineRun, string, error) {
	var baseFn string
	if isObserved {
		baseFn = "observed-pr.yml"
	} else {
		baseFn = "expected-pr.yml"
	}
	fileName := filepath.Join(dir, baseFn)
	exists, err := util.FileExists(fileName)
	if err != nil {
		return nil, fileName, err
	}
	if exists {
		pr := &tektonv1beta1.PipelineRun{}
		data, err := ioutil.ReadFile(fileName)
		if err != nil {
			return nil, fileName, err
		}
		err = yaml.Unmarshal(data, pr)
		if err != nil {
			return nil, fileName, err
		}
		return pr, fileName, err
	}
	return nil, fileName, nil
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
