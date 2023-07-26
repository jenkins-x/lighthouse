package trigger

import (
	"regexp"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/jenkins-x/lighthouse/pkg/filebrowser"
	"github.com/jenkins-x/lighthouse/pkg/jobutil"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/jenkins-x/lighthouse/pkg/triggerconfig/inrepo"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/maps"
	"gopkg.in/robfig/cron.v2"
)

type PeriodicAgent struct {
	Cron      *cron.Cron
	Periodics map[string]map[string]PeriodicExec
}

func (p PeriodicAgent) UpdatePeriodics(org string, repo string, agent plugins.Agent, pe *scm.PushHook) {
	fullName := org + "/" + repo
	// FIXME Here is a race condition with StartPeriodics so that if StartPeriodics and UpdatePeriodics update
	// for a repo at the same time a periodic could be scheduled multiple times
	// Another potential cause for duplicate jobs is that multiple lighthouse webhook processes could run at the same time
	// StartPeriodics is likely so slow though that the risk for missed jobs are greater though
	// Ideally an external lock should be used to synchronise, but it is unlikely to work well. It would probably be better to use something external
	// Maybe integrate with Tekton Triggers, but that would mean another moving part...
	// https://github.com/tektoncd/triggers/tree/main/examples/v1beta1/cron
	// Probably better then to just create CronJobs that create LighthouseJobs/PipelineRuns using kubectl/tkn.
	// Possibly with the Pipeline stored as separate resource and then either have the LighthouseJob refer to it or do tkn pipeline start.
	// With proper labels these CronJobs/Pipelines could be handled fairly efficiently
	// So with LighthouseJobs it could be rendered and put in a configmap which is mounted in the cronjob.
	// Then kubectl apply -f to create the job and then to set the status kubectl patch LighthouseJob myresource --type=merge --subresource status --patch 'status: {state: triggered}'
	// Would really only need to run StartPeriodics when in a new cluster. How do I know when it is needed? I would
	// need to store in cluster when StartPeriodics has been run.
	repoPeriodics := maps.Clone(p.Periodics[fullName])
	if repoPeriodics == nil {
		repoPeriodics = make(map[string]PeriodicExec)
		p.Periodics[fullName] = repoPeriodics
	}
	l := logrus.WithField(scmprovider.RepoLogField, repo).WithField(scmprovider.OrgLogField, org)
	for _, periodic := range agent.Config.Periodics {
		exec, exists := repoPeriodics[periodic.Name]
		if !exists || hasChanged(exec.Periodic, periodic, pe, l) {
			// New or changed. Changed will have old entry removed below
			addPeriodic(l, periodic, org, repo, agent.LauncherClient, p.Cron, fullName, p.Periodics[fullName])
		} else {
			// Nothing to change
			delete(repoPeriodics, periodic.Name)
		}
	}
	// Deschedule periodic no longer found in repo
	for _, periodic := range repoPeriodics {
		p.Cron.Remove(periodic.EntryID)
	}
}

// hasChanged return true if any fields have changed except pipelineLoader and PipelineRunSpec since lazyLoading means you can't compare these
// Also check if the file pointed to by SourcePath has changed
func hasChanged(existing, imported job.Periodic, pe *scm.PushHook, l *logrus.Entry) bool {
	if !cmp.Equal(existing, imported, cmpopts.IgnoreFields(job.Periodic{}, "pipelineLoader", "PipelineRunSpec")) {
		return true
	}
	// Since We don't know which directory SourcePath is relative to we check for any file with that name
	changeMatcher := job.RegexpChangeMatcher{RunIfChanged: regexp.QuoteMeta(existing.SourcePath)}
	_, run, err := changeMatcher.ShouldRun(listPushEventChanges(*pe))
	if err != nil {
		l.WithError(err).Warnf("Can't determine if %s has changed, assumes it hasn't", existing.SourcePath)
	}
	return run
}

type PeriodicExec struct {
	job.Periodic
	Owner, Repo    string
	LauncherClient launcher
	EntryID        cron.EntryID
}

func (p *PeriodicExec) Run() {
	labels := make(map[string]string)
	for k, v := range p.Labels {
		labels[k] = v
	}
	refs := v1alpha1.Refs{
		Org:  p.Owner,
		Repo: p.Repo,
	}
	l := logrus.WithField(scmprovider.RepoLogField, p.Repo).WithField(scmprovider.OrgLogField, p.Owner)

	pj := jobutil.NewLighthouseJob(jobutil.PeriodicSpec(l, p.Periodic, refs), labels, p.Annotations)
	l.WithFields(jobutil.LighthouseJobFields(&pj)).Info("Creating a new LighthouseJob.")
	_, err := p.LauncherClient.Launch(&pj)
	if err != nil {
		l.WithError(err).Error("Failed to create lighthouse job for cron ")
	}
}

func StartPeriodics(configAgent *config.Agent, launcher launcher, fileBrowsers *filebrowser.FileBrowsers, periodicAgent *PeriodicAgent) {
	cronAgent := cron.New()
	periodicAgent.Cron = cronAgent
	periodics := make(map[string]map[string]PeriodicExec)
	periodicAgent.Periodics = periodics
	resolverCache := inrepo.NewResolverCache()
	fc := filebrowser.NewFetchCache()
	c := configAgent.Config()
	for fullName := range c.InRepoConfig.Enabled {
		repoPeriodics := make(map[string]PeriodicExec)
		org, repo := scm.Split(fullName)
		if org == "" {
			logrus.Errorf("Wrong format of %s, not owner/repo", fullName)
			continue
		}
		l := logrus.WithField(scmprovider.RepoLogField, repo).WithField(scmprovider.OrgLogField, org)
		// TODO use github code search to see if any periodics exists before loading?
		// in:file filename:trigger.yaml path:.lighthouse repo:fullName periodics
		// Would need to accommodate for rate limit since only 10 searches per minute are allowed
		// This should make the next TODO less important since not as many clones would be created
		// TODO Ensure that the repo clones are removed and deregistered as soon as possible
		cfg, err := inrepo.LoadTriggerConfig(fileBrowsers, fc, resolverCache, org, repo, "")
		if err != nil {
			l.Error(errors.Wrapf(err, "failed to calculate in repo config"))
			continue
		}

		for _, periodic := range cfg.Spec.Periodics {
			addPeriodic(l, periodic, org, repo, launcher, cronAgent, fullName, repoPeriodics)
		}
		if len(repoPeriodics) > 0 {
			periodics[fullName] = repoPeriodics
		}
	}

	cronAgent.Start()
}

func addPeriodic(l *logrus.Entry, periodic job.Periodic, owner, repo string, launcher launcher, cronAgent *cron.Cron, fullName string, repoPeriodics map[string]PeriodicExec) {
	exec := PeriodicExec{
		Periodic:       periodic,
		Owner:          owner,
		Repo:           repo,
		LauncherClient: launcher,
	}
	var err error
	exec.EntryID, err = cronAgent.AddJob(periodic.Cron, &exec)
	if err != nil {
		l.WithError(err).Errorf("failed to schedule job %s", periodic.Name)
	} else {
		repoPeriodics[periodic.Name] = exec
		l.Infof("Periodic %s is scheduled since it is new or has changed", periodic.Name)
	}
}
