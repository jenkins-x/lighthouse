package tekton

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned/fake"
	lhinformers "github.com/jenkins-x/lighthouse/pkg/client/informers/externalversions"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	tektonfake "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	tektoninformers "github.com/tektoncd/pipeline/pkg/client/informers/externalversions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/yaml"
)

func TestSyncHandler(t *testing.T) {
	testCases := []struct {
		name       string
		inputIsJob bool
	}{
		{
			name:       "start-pullrequest",
			inputIsJob: true,
		},
		{
			name:       "update-job",
			inputIsJob: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			utilrand.Seed(12345)
			testData := path.Join("test_data", "controller", tc.name)
			_, err := os.Stat(testData)
			assert.NoError(t, err)

			observedPR, err := loadControllerPipelineRun(true, testData)
			assert.NoError(t, err)
			observedJob, err := loadLighthouseJob(true, testData)
			assert.NoError(t, err)

			expectedPR, err := loadControllerPipelineRun(false, testData)
			assert.NoError(t, err)
			expectedJob, err := loadLighthouseJob(false, testData)
			assert.NoError(t, err)

			ns := "jx"
			var tektonObjects []runtime.Object
			if observedPR != nil {
				tektonObjects = append(tektonObjects, observedPR)
			}
			var lhObjects []runtime.Object
			if observedJob != nil {
				lhObjects = append(lhObjects, observedJob)
			}

			tektonClient := tektonfake.NewSimpleClientset(tektonObjects...)
			lhClient := fake.NewSimpleClientset(lhObjects...)

			lhInformerFactory := lhinformers.NewSharedInformerFactoryWithOptions(lhClient, time.Minute*30, lhinformers.WithNamespace(ns))

			lhInformer := lhInformerFactory.Lighthouse().V1alpha1().LighthouseJobs()
			lhLister := lhInformer.Lister()

			tektonInformerFactory := tektoninformers.NewSharedInformerFactoryWithOptions(tektonClient, time.Minute*30, tektoninformers.WithNamespace(ns))

			tektonInformer := tektonInformerFactory.Tekton().V1beta1().PipelineRuns()
			tektonLister := tektonInformer.Lister()

			stopCh := context.Background().Done()

			tektonInformerFactory.Start(stopCh)
			lhInformerFactory.Start(stopCh)

			if ok := cache.WaitForCacheSync(stopCh, lhInformer.Informer().HasSynced, tektonInformer.Informer().HasSynced); !ok {
				t.Fatalf("caches never synced")
			}

			controller := &Controller{
				tektonClient: tektonClient,
				lhClient:     lhClient,
				prLister:     tektonLister,
				lhLister:     lhLister,
				logger:       logrus.NewEntry(logrus.StandardLogger()).WithField("controller", controllerName),
				ns:           ns,
			}

			var key string
			if tc.inputIsJob {
				if observedJob != nil {
					key, err = toKey(observedJob)
				} else {
					t.Fatal("Expected an observed LighthouseJob but none loaded from observed-lhjob.yml")
				}
			} else {
				if observedPR != nil {
					key, err = toKey(observedPR)
				} else {
					t.Fatal("Expected an observed PipelineRun but none was loaded from observed-pr.yml")
				}
			}
			assert.NoError(t, err)
			err = controller.syncHandler(key)
			assert.NoError(t, err)

			if expectedPR != nil {
				prs, err := tektonClient.TektonV1beta1().PipelineRuns(ns).List(metav1.ListOptions{})
				assert.NoError(t, err)
				assert.Len(t, prs.Items, 1)
				updatedPR := prs.Items[0].DeepCopy()
				if d := cmp.Diff(expectedPR, updatedPR); d != "" {
					t.Errorf("PipelineRun did not match expected: %s", d)
				}
			}
			if expectedJob != nil {
				jobs, err := lhClient.LighthouseV1alpha1().LighthouseJobs(ns).List(metav1.ListOptions{})
				assert.NoError(t, err)
				assert.Len(t, jobs.Items, 1)
				// Ignore status.starttime since that's always going to be different
				updatedJob := jobs.Items[0].DeepCopy()
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
