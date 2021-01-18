package sync

import (
	"encoding/json"
	"fmt"

	lighthousev1alpha1 "github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/clients"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/jenkins-x/lighthouse/pkg/jobutil"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclient "k8s.io/client-go/kubernetes"
)

const (
	periodicLabel      = "lighthouse.jenkins.io/periodic"
	periodicLabelValue = "true"
)

var periodicSelector = periodicLabel + "=" + periodicLabelValue

// GetCommand creates the cobra command for syncing jobs
func GetCommand() *cobra.Command {
	var namespace string
	var configPath string
	var serviceAccount string
	var tag string

	cmd := &cobra.Command{
		Use: "sync",
		Run: func(cmd *cobra.Command, args []string) {
			execute(namespace, configPath, serviceAccount, tag)
		},
	}
	cmd.Flags().StringVar(&namespace, "namespace", "", "The namespace to create cronjobs in")
	cmd.Flags().StringVar(&configPath, "config-path", "", "Path to config.yaml.")
	cmd.Flags().StringVar(&serviceAccount, "service-account", "", "Service account.")
	cmd.Flags().StringVar(&tag, "tag", "latest", "Strobe docker image tag.")
	if err := cmd.MarkFlagRequired("namespace"); err != nil {
		logrus.WithError(err).Fatal("Failed to init sync command")
	}
	if err := cmd.MarkFlagRequired("config-path"); err != nil {
		logrus.WithError(err).Fatal("Failed to init sync command")
	}
	if err := cmd.MarkFlagRequired("service-account"); err != nil {
		logrus.WithError(err).Fatal("Failed to init sync command")
	}
	return cmd
}

func execute(namespace, configPath, serviceAccount, tag string) {
	logrus.Infof("Syncing cronjob in %s...", namespace)
	lhConfig, err := config.Load(configPath, "")
	if err != nil {
		logrus.WithError(err).Fatal("Error loading config")
	}
	cfg, err := clients.GetConfig("", "")
	if err != nil {
		logrus.WithError(err).Fatal("Could not create kube config")
	}
	k8sClient, err := kubeclient.NewForConfig(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("Could not create kube client")
	}
	if err := updateCronjobs(k8sClient, namespace, serviceAccount, lhConfig.JobConfig, tag); err != nil {
		logrus.WithError(err).Fatal("Could not update cron jobs")
	}
}

func updateCronjobs(k8sClient kubeclient.Interface, namespace string, serviceAccount string, jobConfig job.Config, tag string) error {
	observed := map[string]v1beta1.CronJob{}
	actual, err := k8sClient.BatchV1beta1().CronJobs(namespace).List(metav1.ListOptions{LabelSelector: periodicSelector})
	if err != nil {
		return err
	}
	for _, cronjob := range actual.Items {
		observed[cronjob.Name] = cronjob
	}
	for _, periodic := range jobConfig.Periodics {
		cronjob, err := makeCronJob(namespace, serviceAccount, periodic, tag)
		if err != nil {
			return err
		}
		logrus.Infof("Syncing cronjob %s", cronjob.Name)
		if _, ok := observed[cronjob.Name]; !ok {
			logrus.Infof("Creating cronjob %s", cronjob.Name)
			// create
			_, err := k8sClient.BatchV1beta1().CronJobs(namespace).Create(cronjob)
			if err != nil {
				return err
			}
		} else {
			logrus.Infof("Updating cronjob %s", cronjob.Name)
			// update
			_, err := k8sClient.BatchV1beta1().CronJobs(namespace).Update(cronjob)
			if err != nil {
				return err
			}
		}
	}
	lookup := func(cronjob v1beta1.CronJob) bool {
		for _, periodic := range jobConfig.Periodics {
			if cronjob.Name == periodic.Name {
				return true
			}
		}
		return false
	}
	for _, cronjob := range observed {
		if !lookup(cronjob) {
			logrus.Infof("Deleting cronjob %s", cronjob.Name)
			// delete
			err := k8sClient.BatchV1beta1().CronJobs(namespace).Delete(cronjob.Name, nil)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func makeCronJob(namespace string, serviceAccount string, periodic job.Periodic, tag string) (*v1beta1.CronJob, error) {
	var one int32 = 1
	lhjob := lighthousev1alpha1.LighthouseJob{
		TypeMeta: metav1.TypeMeta{
			Kind:       "LighthouseJob",
			APIVersion: "lighthouse.jenkins.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: periodic.Name + "-",
			Namespace:    namespace,
		},
		Spec: jobutil.PeriodicSpec(periodic),
	}
	b, err := json.Marshal(lhjob)
	if err != nil {
		return nil, err
	}
	labels := map[string]string{}
	for k, v := range periodic.Labels {
		labels[k] = v
	}
	labels[periodicLabel] = periodicLabelValue
	return &v1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:        periodic.Name,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: periodic.Annotations,
		},
		Spec: v1beta1.CronJobSpec{
			Schedule:                   periodic.Cron,
			ConcurrencyPolicy:          v1beta1.ForbidConcurrent,
			SuccessfulJobsHistoryLimit: &one,
			FailedJobsHistoryLimit:     &one,
			JobTemplate: v1beta1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							ServiceAccountName: serviceAccount,
							RestartPolicy:      corev1.RestartPolicyNever,
							Containers: []corev1.Container{{
								Name:  "strobe-start",
								Image: fmt.Sprintf("gcr.io/jenkinsxio/lighthouse-strobe:%s", tag),
								Args: []string{
									"start",
								},
								Env: []corev1.EnvVar{
									{
										Name:  "LHJOB",
										Value: string(b),
									},
								},
							}},
						},
					},
				},
			},
		},
	}, nil
}
