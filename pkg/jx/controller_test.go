package jx

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	jxv1 "github.com/jenkins-x/jx-api/pkg/apis/jenkins.io/v1"
	jxclient "github.com/jenkins-x/jx-api/pkg/client/clientset/versioned"
	jxfake "github.com/jenkins-x/jx-api/pkg/client/clientset/versioned/fake"
	jxinformers "github.com/jenkins-x/jx-api/pkg/client/informers/externalversions"
	"github.com/jenkins-x/jx/v2/pkg/gits"
	"github.com/jenkins-x/jx/v2/pkg/kube"
	"github.com/jenkins-x/jx/v2/pkg/tekton"
	"github.com/jenkins-x/jx/v2/pkg/tekton/metapipeline"
	"github.com/jenkins-x/jx/v2/pkg/util"
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned/fake"
	lhinformers "github.com/jenkins-x/lighthouse/pkg/client/informers/externalversions"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/yaml"
)

type fakeMetapipelineClient struct {
	jxClient jxclient.Interface

	ns string
}

// Create just creates a PipelineActivity key
func (f *fakeMetapipelineClient) Create(param metapipeline.PipelineCreateParam) (kube.PromoteStepActivityKey, tekton.CRDWrapper, error) {
	gitInfo, err := gits.ParseGitURL(param.PullRef.SourceURL())
	if err != nil {
		return kube.PromoteStepActivityKey{}, tekton.CRDWrapper{}, errors.Wrap(err, fmt.Sprintf("unable to determine needed git info from the specified git url '%s'", param.PullRef.SourceURL()))
	}

	var branchIdentifier string
	switch param.PipelineKind {
	case metapipeline.ReleasePipeline:
		// no pull requests to merge, taking base branch name as identifier
		branchIdentifier = param.PullRef.BaseBranch()
	case metapipeline.PullRequestPipeline:
		if len(param.PullRef.PullRequests()) == 0 {
			return kube.PromoteStepActivityKey{}, tekton.CRDWrapper{}, errors.New("pullrequest pipeline requested, but no pull requests specified")
		}
		branchIdentifier = fmt.Sprintf("PR-%s", param.PullRef.PullRequests()[0].ID)
	default:
		branchIdentifier = "unknown"
	}

	pr, _ := tekton.ParsePullRefs(param.PullRef.String())
	pipelineActivity := tekton.GeneratePipelineActivity("1", branchIdentifier, gitInfo, param.Context, pr)

	return *pipelineActivity, tekton.CRDWrapper{}, nil
}

// Apply just applies the PipelineActivity
func (f *fakeMetapipelineClient) Apply(pipelineActivity kube.PromoteStepActivityKey, crds tekton.CRDWrapper) error {
	_, _, err := pipelineActivity.GetOrCreate(f.jxClient, f.ns)
	if err != nil {
		return err
	}
	return nil
}

// Close is a no-op here
func (f *fakeMetapipelineClient) Close() error {
	return nil
}

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
		{
			name:       "no-job-for-activity",
			inputIsJob: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testData := path.Join("test_data", "controller", tc.name)
			_, err := os.Stat(testData)
			assert.NoError(t, err)

			observedActivity, err := loadPipelineActivity(true, testData)
			assert.NoError(t, err)
			observedJob, err := loadLighthouseJob(true, testData)
			assert.NoError(t, err)

			expectedActivity, err := loadPipelineActivity(false, testData)
			assert.NoError(t, err)
			expectedJob, err := loadLighthouseJob(false, testData)
			assert.NoError(t, err)

			ns := "jx"
			var jxObjects []runtime.Object
			if observedActivity != nil {
				jxObjects = append(jxObjects, observedActivity)
			}
			var lhObjects []runtime.Object
			if observedJob != nil {
				lhObjects = append(lhObjects, observedJob)
			}

			jxClient := jxfake.NewSimpleClientset(jxObjects...)
			lhClient := fake.NewSimpleClientset(lhObjects...)

			lhInformerFactory := lhinformers.NewSharedInformerFactoryWithOptions(lhClient, time.Minute*30, lhinformers.WithNamespace(ns))

			lhInformer := lhInformerFactory.Lighthouse().V1alpha1().LighthouseJobs()
			lhLister := lhInformer.Lister()

			jxInformerFactory := jxinformers.NewSharedInformerFactoryWithOptions(jxClient, time.Minute*30, jxinformers.WithNamespace(ns))

			jxInformer := jxInformerFactory.Jenkins().V1().PipelineActivities()
			jxLister := jxInformer.Lister()

			stopCh := context.Background().Done()

			jxInformerFactory.Start(stopCh)
			lhInformerFactory.Start(stopCh)

			if ok := cache.WaitForCacheSync(stopCh, lhInformer.Informer().HasSynced, jxInformer.Informer().HasSynced); !ok {
				t.Fatalf("caches never synced")
			}
			mpc := &fakeMetapipelineClient{
				jxClient: jxClient,
				ns:       ns,
			}

			controller := &Controller{
				jxClient:       jxClient,
				lhClient:       lhClient,
				mpClient:       mpc,
				activityLister: jxLister,
				lhLister:       lhLister,
				logger:         logrus.NewEntry(logrus.StandardLogger()).WithField("controller", controllerName),
				ns:             ns,
			}

			var key string
			if tc.inputIsJob {
				if observedJob != nil {
					key, err = toKey(observedJob)
				} else {
					t.Fatal("Expected an observed LighthouseJob but none loaded from observed-lhjob.yml")
				}
			} else {
				if observedActivity != nil {
					key, err = toKey(observedActivity)
				} else {
					t.Fatal("Expected an observed PipelineActivity but none was loaded from observed-activity.yml")
				}
			}
			assert.NoError(t, err)
			err = controller.syncHandler(key)
			assert.NoError(t, err)

			if expectedActivity != nil {
				activities, err := jxClient.JenkinsV1().PipelineActivities(ns).List(metav1.ListOptions{})
				assert.NoError(t, err)
				assert.Len(t, activities.Items, 1)
				updatedActivity := activities.Items[0].DeepCopy()
				if d := cmp.Diff(expectedActivity, updatedActivity); d != "" {
					t.Errorf("PipelineActivity did not match expected: %s", d)
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

func loadPipelineActivity(isObserved bool, dir string) (*jxv1.PipelineActivity, error) {
	var baseFn string
	if isObserved {
		baseFn = "observed-activity.yml"
	} else {
		baseFn = "expected-activity.yml"
	}
	fileName := filepath.Join(dir, baseFn)
	exists, err := util.FileExists(fileName)
	if err != nil {
		return nil, err
	}
	if exists {
		activity := &jxv1.PipelineActivity{}
		data, err := ioutil.ReadFile(fileName)
		if err != nil {
			return nil, err
		}
		err = yaml.Unmarshal(data, activity)
		if err != nil {
			return nil, err
		}
		return activity, err
	}
	return nil, nil
}
