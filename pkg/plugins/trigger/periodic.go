package trigger

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/jenkins-x/lighthouse/pkg/filebrowser"
	"github.com/jenkins-x/lighthouse/pkg/jobutil"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/jenkins-x/lighthouse/pkg/triggerconfig/inrepo"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	applybatchv1 "k8s.io/client-go/applyconfigurations/batch/v1"
	applyv1 "k8s.io/client-go/applyconfigurations/core/v1"
	kubeclient "k8s.io/client-go/kubernetes"
	typedbatchv1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	typedv1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type PeriodicAgent struct {
	Namespace string
	SCMClient *scm.Client
}

const fieldManager = "lighthouse"
const initializedField = "isPeriodicsInitialized"
const initStartedField = "periodicsInitializationStarted"

func (pa *PeriodicAgent) UpdatePeriodics(kc kubeclient.Interface, agent plugins.Agent, pe *scm.PushHook) {
	repo := pe.Repository()
	l := logrus.WithField(scmprovider.RepoLogField, repo.Name).WithField(scmprovider.OrgLogField, repo.Namespace)
	if !hasChanges(pe, agent) {
		return
	}
	cmInterface := kc.CoreV1().ConfigMaps(pa.Namespace)
	cjInterface := kc.BatchV1().CronJobs(pa.Namespace)
	cmList, cronList, done := pa.getExistingResources(l, cmInterface, cjInterface,
		fmt.Sprintf("app=lighthouse-webhooks,component=periodic,org=%s,repo=%s,trigger", repo.Namespace, repo.Name))
	if done {
		return
	}

	getExistingConfigMap := func(p job.Periodic) *corev1.ConfigMap {
		for i, cm := range cmList.Items {
			if cm.Labels["trigger"] == p.Name {
				cmList.Items[i] = corev1.ConfigMap{}
				return &cm
			}
		}
		return nil
	}
	getExistingCron := func(p job.Periodic) *batchv1.CronJob {
		for i, cj := range cronList.Items {
			if cj.Labels["trigger"] == p.Name {
				cronList.Items[i] = batchv1.CronJob{}
				return &cj
			}
		}
		return nil
	}

	if pa.UpdatePeriodicsForRepo(
		agent.Config.Periodics,
		l,
		getExistingConfigMap,
		getExistingCron,
		repo.Namespace,
		repo.Name,
		cmInterface,
		cjInterface,
	) {
		return
	}

	for _, cj := range cronList.Items {
		if cj.Name != "" {
			deleteCronJob(cjInterface, &cj)
		}
	}
	for _, cm := range cmList.Items {
		if cm.Name != "" {
			deleteConfigMap(cmInterface, &cm)
		}
	}
}

// hasChanges return true if any triggers.yaml or file pointed to by SourcePath has changed
func hasChanges(pe *scm.PushHook, agent plugins.Agent) bool {
	changedFiles, err := listPushEventChanges(*pe)()
	if err != nil {
		return false
	}
	lighthouseFiles := make(map[string]bool)
	for _, changedFile := range changedFiles {
		if strings.HasPrefix(changedFile, ".lighthouse/") {
			_, changedFile = filepath.Split(changedFile)
			if changedFile == "triggers.yaml" {
				return true
			}
			lighthouseFiles[changedFile] = true
		}
	}
	for _, p := range agent.Config.Periodics {
		_, sourcePath := filepath.Split(p.SourcePath)
		if lighthouseFiles[sourcePath] {
			return true
		}
	}
	return false
}

func (pa *PeriodicAgent) PeriodicsInitialized(namespace string, kc kubeclient.Interface) bool {
	cmInterface := kc.CoreV1().ConfigMaps(namespace)
	cm, err := cmInterface.Get(context.TODO(), util.ProwConfigMapName, metav1.GetOptions{})
	if err != nil {
		logrus.Errorf("Can't get ConfigMap config. Can't check if periodics as initialized")
		return true
	}
	isInit := cm.Data[initializedField]
	if "true" == isInit {
		return true
	}
	if isInit == "pending" {
		initStarted, err := strconv.ParseInt(cm.Data[initStartedField], 10, 64)
		// If started less than 24 hours ago we assume it still goes on so return true
		if err == nil && time.Unix(initStarted, 0).Before(time.Now().Add(-24*time.Hour)) {
			return true
		}
	}
	cmApply, err := applyv1.ExtractConfigMap(cm, fieldManager)
	cmApply.Data[initializedField] = "pending"
	cm.Data[initStartedField] = strconv.FormatInt(time.Now().Unix(), 10)
	_, err = cmInterface.Apply(context.TODO(), cmApply, metav1.ApplyOptions{FieldManager: "lighthouse"})
	if err != nil {
		// Somebody else has updated the configmap, so don't initialize periodics now
		return true
	}
	return false
}

func (pa *PeriodicAgent) InitializePeriodics(kc kubeclient.Interface, configAgent *config.Agent, fileBrowsers *filebrowser.FileBrowsers) {
	if pa.SCMClient == nil {
		_, scmClient, _, _, err := util.GetSCMClient("", configAgent.Config)
		if err != nil {
			logrus.Errorf("failed to create SCM scmClient: %s", err.Error())
			return
		}
		pa.SCMClient = scmClient
	}

	resolverCache := inrepo.NewResolverCache()
	fc := filebrowser.NewFetchCache()
	c := configAgent.Config()
	cmInterface := kc.CoreV1().ConfigMaps(pa.Namespace)
	cjInterface := kc.BatchV1().CronJobs(pa.Namespace)
	cmList, cronList, done := pa.getExistingResources(nil, cmInterface, cjInterface, "app=lighthouse-webhooks,component=periodic,org,repo,trigger")
	if done {
		return
	}
	cmMap := make(map[string]map[string]*corev1.ConfigMap)
	for _, cm := range cmList.Items {
		cmMap[cm.Labels["org"]+"/"+cm.Labels["repo"]][cm.Labels["trigger"]] = &cm
	}
	cronMap := make(map[string]map[string]*batchv1.CronJob)
	for _, cronjob := range cronList.Items {
		cronMap[cronjob.Labels["org"]+"/"+cronjob.Labels["repo"]][cronjob.Labels["trigger"]] = &cronjob
	}

	for fullName := range pa.filterPeriodics(c.InRepoConfig.Enabled) {
		repoCronJobs, repoCronExists := cronMap[fullName]
		repoCM, repoCmExists := cmMap[fullName]
		org, repo := scm.Split(fullName)
		if org == "" {
			logrus.Errorf("Wrong format of %s, not owner/repo", fullName)
			continue
		}
		l := logrus.WithField(scmprovider.RepoLogField, repo).WithField(scmprovider.OrgLogField, org)
		// TODO Ensure that the repo clones are removed and deregistered as soon as possible
		// One solution would be to run InitializePeriodics in a separate job
		cfg, err := inrepo.LoadTriggerConfig(fileBrowsers, fc, resolverCache, org, repo, "")
		if err != nil {
			l.Error(errors.Wrapf(err, "failed to calculate in repo config"))
			// Keeping existing cronjobs if trigger config can not be read
			delete(cronMap, fullName)
			delete(cmMap, fullName)
			continue
		}
		getExistingCron := func(p job.Periodic) *batchv1.CronJob {
			if repoCronExists {
				cj, cjExists := repoCronJobs[p.Name]
				if cjExists {
					delete(repoCronJobs, p.Name)
					return cj
				}
			}
			return nil
		}

		getExistingConfigMap := func(p job.Periodic) *corev1.ConfigMap {
			if repoCmExists {
				cm, cmExist := repoCM[p.Name]
				if cmExist {
					delete(repoCM, p.Name)
					return cm
				}
			}
			return nil
		}

		if pa.UpdatePeriodicsForRepo(cfg.Spec.Periodics, l, getExistingConfigMap, getExistingCron, org, repo, cmInterface, cjInterface) {
			return
		}
	}

	// Removing CronJobs not corresponding to any found triggers
	for _, repoCron := range cronMap {
		for _, aCron := range repoCron {
			deleteCronJob(cjInterface, aCron)
		}
	}
	for _, repoCm := range cmMap {
		for _, cm := range repoCm {
			deleteConfigMap(cmInterface, cm)
		}
	}
	cmInterface.Apply(context.TODO(),
		(&applyv1.ConfigMapApplyConfiguration{}).
			WithName("config").
			WithData(map[string]string{initializedField: "true"}),
		metav1.ApplyOptions{Force: true, FieldManager: "lighthouse"})
}

func deleteConfigMap(cmInterface typedv1.ConfigMapInterface, cm *corev1.ConfigMap) {
	err := cmInterface.Delete(context.TODO(), cm.Name, metav1.DeleteOptions{})
	if err != nil {
		logrus.WithError(err).
			Errorf("Failed to delete ConfigMap %s corresponding to removed trigger %s for repo %s/%s",
				cm.Name, cm.Labels["trigger"], cm.Labels["org"], cm.Labels["repo"])
	}
}

func deleteCronJob(cjInterface typedbatchv1.CronJobInterface, cj *batchv1.CronJob) {
	err := cjInterface.Delete(context.TODO(), cj.Name, metav1.DeleteOptions{})
	if err != nil {
		logrus.WithError(err).
			Errorf("Failed to delete CronJob %s corresponding to removed trigger %s for repo %s",
				cj.Name, cj.Labels["trigger"], cj.Labels["repo"])
	}
}

func (pa *PeriodicAgent) UpdatePeriodicsForRepo(
	periodics []job.Periodic,
	l *logrus.Entry,
	getExistingConfigMap func(p job.Periodic) *corev1.ConfigMap,
	getExistingCron func(p job.Periodic) *batchv1.CronJob,
	org string,
	repo string,
	cmInterface typedv1.ConfigMapInterface,
	cjInterface typedbatchv1.CronJobInterface,
) bool {
	for _, p := range periodics {
		labels := map[string]string{
			"app":       "lighthouse-webhooks",
			"component": "periodic",
			"org":       org,
			"repo":      repo,
			"trigger":   p.Name,
		}
		for k, v := range p.Labels {
			// don't overwrite labels since that would disturb the logic
			_, predef := labels[k]
			if !predef {
				labels[k] = v
			}
		}

		resourceName := fmt.Sprintf("lighthouse-%s-%s-%s", org, repo, p.Name)

		err := p.LoadPipeline(l)
		if err != nil {
			l.WithError(err).Warnf("Failed to load pipeline %s from %s", p.Name, p.SourcePath)
			continue
		}
		refs := v1alpha1.Refs{
			Org:  org,
			Repo: repo,
		}

		pj := jobutil.NewLighthouseJob(jobutil.PeriodicSpec(l, p, refs), labels, p.Annotations)
		lighthouseData, err := json.Marshal(pj)

		// Only apply if any value have changed
		existingCm := getExistingConfigMap(p)

		if existingCm == nil || existingCm.Data["lighthousejob.yaml"] != string(lighthouseData) {
			var cm *applyv1.ConfigMapApplyConfiguration
			if existingCm != nil {
				cm, err = applyv1.ExtractConfigMap(existingCm, fieldManager)
				if err != nil {
					l.Error(errors.Wrapf(err, "failed to extract ConfigMap"))
					return true
				}
			} else {
				cm = (&applyv1.ConfigMapApplyConfiguration{}).WithName(resourceName).WithLabels(labels)
			}
			if cm.Data == nil {
				cm.Data = make(map[string]string)
			}
			cm.Data["lighthousejob.yaml"] = string(lighthouseData)

			_, err := cmInterface.Apply(context.TODO(), cm, metav1.ApplyOptions{Force: true, FieldManager: fieldManager})
			if err != nil {
				l.WithError(err).Errorf("failed to apply configmap")
				return false
			}
		}
		existingCron := getExistingCron(p)
		if existingCron == nil || existingCron.Spec.Schedule != p.Cron {
			var cj *applybatchv1.CronJobApplyConfiguration
			if existingCron != nil {
				cj, err = applybatchv1.ExtractCronJob(existingCron, fieldManager)
				if err != nil {
					l.Error(errors.Wrapf(err, "failed to extract CronJob"))
					return true
				}
			} else {
				cj = pa.constructCronJob(resourceName, resourceName, labels)
			}
			cj.Spec.Schedule = &p.Cron
			_, err := cjInterface.Apply(context.TODO(), cj, metav1.ApplyOptions{Force: true, FieldManager: fieldManager})
			if err != nil {
				l.WithError(err).Errorf("failed to apply cronjob")
				return false
			}
		}
	}
	return false
}

func (pa *PeriodicAgent) getExistingResources(
	l *logrus.Entry,
	cmInterface typedv1.ConfigMapInterface,
	cjInterface typedbatchv1.CronJobInterface,
	selector string,
) (*corev1.ConfigMapList, *batchv1.CronJobList, bool) {
	cmList, err := cmInterface.List(context.TODO(), metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		l.Error("Can't get periodic ConfigMaps. Periodics will not be initialized", err)
		return nil, nil, true
	}

	cronList, err := cjInterface.List(context.TODO(), metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		l.Error("Can't get periodic CronJobs. Periodics will not be initialized", err)
		return nil, nil, true
	}
	return cmList, cronList, false
}

func (pa *PeriodicAgent) constructCronJob(resourceName, configMapName string, labels map[string]string) *applybatchv1.CronJobApplyConfiguration {
	const volumeName = "ligthousejob"
	serviceAccount, found := os.LookupEnv("SERVICE_ACCOUNT")
	if !found {
		serviceAccount = "lighthouse-webhooks"
	}
	return (&applybatchv1.CronJobApplyConfiguration{}).
		WithName(resourceName).
		WithLabels(labels).
		WithSpec((&applybatchv1.CronJobSpecApplyConfiguration{}).
			WithJobTemplate((&applybatchv1.JobTemplateSpecApplyConfiguration{}).
				WithLabels(labels).
				WithSpec((&applybatchv1.JobSpecApplyConfiguration{}).
					WithBackoffLimit(0).
					WithTemplate((&applyv1.PodTemplateSpecApplyConfiguration{}).
						WithLabels(labels).
						WithSpec((&applyv1.PodSpecApplyConfiguration{}).
							WithEnableServiceLinks(false).
							WithServiceAccountName(serviceAccount).
							WithRestartPolicy("Never").
							WithContainers((&applyv1.ContainerApplyConfiguration{}).
								WithName("create-lighthousejob").
								WithImage("bitnami/kubectl").
								WithCommand("/bin/sh").
								WithArgs("-c", `
set -o errexit
create_output=$(kubectl create -f /config/lighthousejob.yaml)
[[ $create_output =~ (.*)\  ]]
kubectl patch ${BASH_REMATCH[1]} --type=merge --subresource status --patch 'status: {state: triggered}'
`).
								WithVolumeMounts((&applyv1.VolumeMountApplyConfiguration{}).
									WithName(volumeName).
									WithMountPath("/config"))).
							WithVolumes((&applyv1.VolumeApplyConfiguration{}).
								WithName(volumeName).
								WithConfigMap((&applyv1.ConfigMapVolumeSourceApplyConfiguration{}).
									WithName(configMapName))))))))
}

func (pa *PeriodicAgent) filterPeriodics(enabled map[string]*bool) map[string]*bool {
	if pa.SCMClient.Contents == nil {
		return enabled
	}

	enable := true
	hasPeriodics := make(map[string]*bool)
	for fullName := range enabled {
		list, _, err := pa.SCMClient.Contents.List(context.TODO(), fullName, ".lighthouse", "HEAD")
		if err != nil {
			continue
		}
		for _, file := range list {
			if file.Type == "dir" {
				triggers, _, err := pa.SCMClient.Contents.Find(context.TODO(), fullName, file.Path+"/triggers.yaml", "HEAD")
				if err != nil {
					continue
				}
				if strings.Contains(string(triggers.Data), "periodics:") {
					hasPeriodics[fullName] = &enable
				}
			}
		}
		delayForRate(pa.SCMClient.Rate())
	}

	return hasPeriodics
}

func delayForRate(r scm.Rate) {
	if r.Remaining < 100 {
		duration := time.Duration(r.Reset - time.Now().Unix())
		logrus.Warnf("waiting for %s seconds until rate limit reset: %+v", duration, r)
		time.Sleep(duration * time.Second)
	}
}
