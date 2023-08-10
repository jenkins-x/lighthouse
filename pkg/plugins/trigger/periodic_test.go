package trigger

import (
	"context"
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	scmfake "github.com/jenkins-x/go-scm/scm/driver/fake"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/config/lighthouse"
	"github.com/jenkins-x/lighthouse/pkg/filebrowser"
	fbfake "github.com/jenkins-x/lighthouse/pkg/filebrowser/fake"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/triggerconfig/inrepo"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kubefake "k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
)

var kubeClient *kubefake.Clientset

// TODO: test more cases
func TestUpdatePeriodics(t *testing.T) {
	namespace, p := setupPeriodicsTest()
	fileBrowsers, err := filebrowser.NewFileBrowsers(filebrowser.GitHubURL, fbfake.NewFakeFileBrowser("test_data", true))
	resolverCache := inrepo.NewResolverCache()
	fc := filebrowser.NewFetchCache()
	cfg, err := inrepo.LoadTriggerConfig(fileBrowsers, fc, resolverCache, "testorg", "myapp", "")

	agent := plugins.Agent{
		Config: &config.Config{
			JobConfig: config.JobConfig{
				Periodics: cfg.Spec.Periodics,
			},
		},
		Logger: logrus.WithField("plugin", pluginName),
	}

	pe := &scm.PushHook{
		Ref: "refs/heads/master",
		Repo: scm.Repository{
			Namespace: "testorg",
			Name:      "myapp",
			FullName:  "testorg/myapp",
		},
		Commits: []scm.PushCommit{
			{
				ID:      "12345678909876",
				Message: "Adding periodics",
				Modified: []string{
					".lighthouse/jenkins-x/triggers.yaml",
				},
			},
		},
	}

	p.UpdatePeriodics(kubeClient, agent, pe)

	selector := "app=lighthouse-webhooks,component=periodic,repo,trigger"
	cms, err := kubeClient.CoreV1().ConfigMaps(namespace).
		List(context.TODO(), metav1.ListOptions{LabelSelector: selector})
	require.NoError(t, err, "failed to get ConfigMaps")
	require.Len(t, cms.Items, 1)
	require.Equal(t, lighthouseJob, cms.Items[0].Data["lighthousejob.yaml"])

	cjs, err := kubeClient.BatchV1().CronJobs(namespace).List(context.TODO(), metav1.ListOptions{})
	require.NoError(t, err, "failed to get CronJobs")
	require.Len(t, cjs.Items, 1)
	cj := cjs.Items[0].Spec
	require.Equal(t, "0 4 * * MON-FRI", cj.Schedule)
	containers := cj.JobTemplate.Spec.Template.Spec.Containers
	require.Len(t, containers, 1)
	require.Len(t, containers[0].Args, 2)
}

func TestInitializePeriodics(t *testing.T) {
	namespace, p := setupPeriodicsTest()

	var enabled = true
	configAgent := &config.Agent{}
	configAgent.Set(&config.Config{
		ProwConfig: lighthouse.Config{
			InRepoConfig: lighthouse.InRepoConfig{
				Enabled: map[string]*bool{"testorg/myapp": &enabled},
			},
		},
	})
	fileBrowsers, err := filebrowser.NewFileBrowsers(filebrowser.GitHubURL, fbfake.NewFakeFileBrowser("test_data", true))
	require.NoError(t, err, "failed to create filebrowsers")

	p.InitializePeriodics(kubeClient, configAgent, fileBrowsers)

	selector := "app=lighthouse-webhooks,component=periodic,org,repo,trigger"
	cms, err := kubeClient.CoreV1().ConfigMaps(namespace).
		List(context.TODO(), metav1.ListOptions{LabelSelector: selector})
	require.NoError(t, err, "failed to get ConfigMaps")
	require.Len(t, cms.Items, 1)
	require.Equal(t, lighthouseJob, cms.Items[0].Data["lighthousejob.yaml"])

	cjs, err := kubeClient.BatchV1().CronJobs(namespace).List(context.TODO(), metav1.ListOptions{})
	require.NoError(t, err, "failed to get CronJobs")
	require.Len(t, cjs.Items, 1)
	cj := cjs.Items[0].Spec
	require.Equal(t, "0 4 * * MON-FRI", cj.Schedule)
	containers := cj.JobTemplate.Spec.Template.Spec.Containers
	require.Len(t, containers, 1)
	require.Len(t, containers[0].Args, 2)
}

func setupPeriodicsTest() (string, *PeriodicAgent) {
	const namespace = "default"
	newDefault, data := scmfake.NewDefault()
	data.ContentDir = "test_data"

	p := &PeriodicAgent{Namespace: namespace, SCMClient: newDefault}
	kubeClient = kubefake.NewSimpleClientset(&v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "config"},
	})

	kubeClient.PrependReactor(
		"patch",
		"configmaps",
		fakeUpsert,
	)

	kubeClient.PrependReactor(
		"patch",
		"cronjobs",
		fakeUpsert,
	)
	return namespace, p
}

func fakeUpsert(action clienttesting.Action) (handled bool, ret runtime.Object, err error) {
	pa := action.(clienttesting.PatchAction)
	if pa.GetPatchType() == types.ApplyPatchType {
		// Apply patches are supposed to upsert, but fake client fails if the object doesn't exist,
		// if an apply patch occurs for a deployment that doesn't yet exist, create it.
		// However, we already hold the fakeclient lock, so we can't use the front door.
		rfunc := clienttesting.ObjectReaction(kubeClient.Tracker())
		_, obj, err := rfunc(
			clienttesting.NewGetAction(pa.GetResource(), pa.GetNamespace(), pa.GetName()),
		)
		if kerrors.IsNotFound(err) || obj == nil {
			objmeta := metav1.ObjectMeta{
				Name:      pa.GetName(),
				Namespace: pa.GetNamespace(),
			}
			var newobj runtime.Object
			switch pa.GetResource().Resource {
			case "configmaps":
				newobj = &v1.ConfigMap{ObjectMeta: objmeta}
			case "cronjobs":
				newobj = &batchv1.CronJob{ObjectMeta: objmeta}
			}
			_, _, _ = rfunc(
				clienttesting.NewCreateAction(
					pa.GetResource(),
					pa.GetNamespace(),
					newobj,
				),
			)
		}
		return rfunc(clienttesting.NewPatchAction(
			pa.GetResource(),
			pa.GetNamespace(),
			pa.GetName(),
			types.StrategicMergePatchType,
			pa.GetPatch()))
	}
	return false, nil, nil
}

const lighthouseJob = `{"kind":"LighthouseJob","apiVersion":"lighthouse.jenkins.io/v1alpha1","metadata":{"generateName":"testorg-myapp-","creationTimestamp":null,"labels":{"app":"lighthouse-webhooks","component":"periodic","created-by-lighthouse":"true","lighthouse.jenkins-x.io/job":"dailyjob","lighthouse.jenkins-x.io/type":"periodic","org":"testorg","repo":"myapp","trigger":"dailyjob"},"annotations":{"lighthouse.jenkins-x.io/job":"dailyjob"}},"spec":{"type":"periodic","agent":"tekton-pipeline","job":"dailyjob","refs":{"org":"testorg","repo":"myapp"},"pipeline_run_spec":{"pipelineSpec":{"tasks":[{"name":"echo-greeting","taskRef":{"name":"task-echo-message"},"params":[{"name":"MESSAGE","value":"$(params.GREETINGS)"},{"name":"BUILD_ID","value":"$(params.BUILD_ID)"},{"name":"JOB_NAME","value":"$(params.JOB_NAME)"},{"name":"JOB_SPEC","value":"$(params.JOB_SPEC)"},{"name":"JOB_TYPE","value":"$(params.JOB_TYPE)"},{"name":"PULL_BASE_REF","value":"$(params.PULL_BASE_REF)"},{"name":"PULL_BASE_SHA","value":"$(params.PULL_BASE_SHA)"},{"name":"PULL_NUMBER","value":"$(params.PULL_NUMBER)"},{"name":"PULL_PULL_REF","value":"$(params.PULL_PULL_REF)"},{"name":"PULL_PULL_SHA","value":"$(params.PULL_PULL_SHA)"},{"name":"PULL_REFS","value":"$(params.PULL_REFS)"},{"name":"REPO_NAME","value":"$(params.REPO_NAME)"},{"name":"REPO_OWNER","value":"$(params.REPO_OWNER)"},{"name":"REPO_URL","value":"$(params.REPO_URL)"}]}],"params":[{"name":"GREETINGS","type":"string","description":"morning greetings, default is Good Morning!","default":"Good Morning!"},{"name":"BUILD_ID","type":"string","description":"the unique build number"},{"name":"JOB_NAME","type":"string","description":"the name of the job which is the trigger context name"},{"name":"JOB_SPEC","type":"string","description":"the specification of the job"},{"name":"JOB_TYPE","type":"string","description":"'the kind of job: postsubmit or presubmit'"},{"name":"PULL_BASE_REF","type":"string","description":"the base git reference of the pull request"},{"name":"PULL_BASE_SHA","type":"string","description":"the git sha of the base of the pull request"},{"name":"PULL_NUMBER","type":"string","description":"git pull request number","default":""},{"name":"PULL_PULL_REF","type":"string","description":"git pull request ref in the form 'refs/pull/$PULL_NUMBER/head'","default":""},{"name":"PULL_PULL_SHA","type":"string","description":"git revision to checkout (branch, tag, sha, ref…)","default":""},{"name":"PULL_REFS","type":"string","description":"git pull reference strings of base and latest in the form 'master:$PULL_BASE_SHA,$PULL_NUMBER:$PULL_PULL_SHA:refs/pull/$PULL_NUMBER/head'"},{"name":"REPO_NAME","type":"string","description":"git repository name"},{"name":"REPO_OWNER","type":"string","description":"git repository owner (user or organisation)"},{"name":"REPO_URL","type":"string","description":"git url to clone"}]}},"pipeline_run_params":[{"name":"GREETINGS"}]},"status":{"startTime":null}}`
