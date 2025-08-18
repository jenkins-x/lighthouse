/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package keeper contains a controller for managing a keeper pool of PRs. The
// controller will automatically retest PRs in the pool and merge them if they
// pass tests.
package keeper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	clientset "github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/jenkins-x/lighthouse/pkg/config/keeper"
	"github.com/jenkins-x/lighthouse/pkg/errorutil"
	"github.com/jenkins-x/lighthouse/pkg/filebrowser"
	"github.com/jenkins-x/lighthouse/pkg/git"
	"github.com/jenkins-x/lighthouse/pkg/jobutil"
	"github.com/jenkins-x/lighthouse/pkg/keeper/blockers"
	"github.com/jenkins-x/lighthouse/pkg/keeper/history"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/jenkins-x/lighthouse/pkg/triggerconfig/inrepo"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	githubql "github.com/shurcooL/githubv4"
	"github.com/sirupsen/logrus"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

// For mocking out sleep during unit tests.
var sleep = time.Sleep

type launcher interface {
	Launch(*v1alpha1.LighthouseJob) (*v1alpha1.LighthouseJob, error)
}

type scmProviderClient interface {
	CreateGraphQLStatus(string, string, string, *scmprovider.Status) (*scm.Status, error)
	GetCombinedStatus(org, repo, ref string) (*scm.CombinedStatus, error)
	CreateStatus(org, repo, ref string, s *scm.StatusInput) (*scm.Status, error)
	GetPullRequestChanges(org, repo string, number int) ([]*scm.Change, error)
	ListPullRequestComments(owner, repo string, number int) ([]*scm.Comment, error)
	GetRef(string, string, string) (string, error)
	Merge(string, string, int, scmprovider.MergeDetails) error
	Query(context.Context, interface{}, map[string]interface{}) error
	SupportsGraphQL() bool
	ProviderType() string
	PRRefFmt() string
	GetRepositoryByFullName(string) (*scm.Repository, error)
	ListAllPullRequestsForFullNameRepo(string, scm.PullRequestListOptions) ([]*scm.PullRequest, error)
	CreateComment(owner, repo string, number int, isPR bool, comment string) error
	EditComment(owner, repo string, number int, id int, comment string, pr bool) error
	GetFile(string, string, string, string) ([]byte, error)
	ListFiles(string, string, string, string) ([]*scm.FileEntry, error)
	GetIssueLabels(string, string, int, bool) ([]*scm.Label, error)
}

type contextChecker interface {
	// IsOptional tells whether a context is optional.
	IsOptional(string) bool
	// MissingRequiredContexts tells if required contexts are missing from the list of contexts provided.
	MissingRequiredContexts([]string) []string
}

// DefaultController knows how to sync PRs and PJs.
type DefaultController struct {
	logger         *logrus.Entry
	config         config.Getter
	spc            scmProviderClient
	fileBrowsers   *filebrowser.FileBrowsers
	launcherClient launcher
	gc             git.Client
	tektonClient   tektonclient.Interface
	lhClient       clientset.Interface
	ns             string

	sc *statusController

	m     sync.Mutex
	pools []Pool

	// changedFiles caches the names of files changed by PRs.
	// Cache entries expire if they are not used during a sync loop.
	changedFiles *changedFilesAgent

	History *history.History
}

// Action represents what actions the controller can take. It will take
// exactly one action each sync.
type Action string

// Constants for various actions the controller might take
const (
	Wait         Action = "WAIT"
	Trigger      Action = "TRIGGER"
	TriggerBatch Action = "TRIGGER_BATCH"
	Merge        Action = "MERGE"
	MergeBatch   Action = "MERGE_BATCH"
	PoolBlocked  Action = "BLOCKED"
)

// recordableActions is the subset of actions that we keep historical record of.
// Ignore idle actions to avoid flooding the records with useless data.
var recordableActions = map[Action]bool{
	Trigger:      true,
	TriggerBatch: true,
	Merge:        true,
	MergeBatch:   true,
}

// Pool represents information about a keeper pool. There is one for every
// org/repo/branch combination that has PRs in the pool.
type Pool struct {
	Org    string
	Repo   string
	Branch string

	// PRs with passing tests, pending tests, and missing or failed tests.
	// Note that these results are rolled up. If all tests for a PR are passing
	// except for one pending, it will be in PendingPRs.
	SuccessPRs []PullRequest
	PendingPRs []PullRequest
	MissingPRs []PullRequest

	// Empty if there is no pending batch.
	BatchPending []PullRequest

	// Which action did we last take, and to what target(s), if any.
	Action   Action
	Target   []PullRequest
	Blockers []blockers.Blocker
	Error    string
}

type prWithStatus struct {
	pr              PullRequest
	success         bool
	waitingFor      []int
	waitingForBatch []int
	blocks          []blockers.Blocker
}

// Prometheus Metrics
var (
	keeperMetrics = struct {
		// Per pool
		pooledPRs  *prometheus.GaugeVec
		updateTime *prometheus.GaugeVec
		merges     *prometheus.HistogramVec

		// Singleton
		syncDuration         prometheus.Gauge
		statusUpdateDuration prometheus.Gauge
	}{
		pooledPRs: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "pooledprs",
			Help: "Number of PRs in each Keeper pool.",
		}, []string{
			"org",
			"repo",
			"branch",
		}),
		updateTime: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "updatetime",
			Help: "The last time each subpool was synced. (Used to determine 'pooledprs' freshness.)",
		}, []string{
			"org",
			"repo",
			"branch",
		}),

		merges: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "merges",
			Help:    "Histogram of merges where values are the number of PRs merged together.",
			Buckets: []float64{1, 2, 3, 4, 5, 7, 10, 15, 25},
		}, []string{
			"org",
			"repo",
			"branch",
		}),

		syncDuration: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "syncdur",
			Help: "The duration of the last loop of the sync controller.",
		}),

		statusUpdateDuration: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "statusupdatedur",
			Help: "The duration of the last loop of the status update controller.",
		}),
	}
)

func init() {
	prometheus.MustRegister(keeperMetrics.pooledPRs)
	prometheus.MustRegister(keeperMetrics.updateTime)
	prometheus.MustRegister(keeperMetrics.merges)
	prometheus.MustRegister(keeperMetrics.syncDuration)
	prometheus.MustRegister(keeperMetrics.statusUpdateDuration)
}

// NewController makes a DefaultController out of the given clients.
func NewController(spcSync, spcStatus *scmprovider.Client, fileBrowsers *filebrowser.FileBrowsers, launcherClient launcher, tektonClient tektonclient.Interface, lighthouseClient clientset.Interface, ns string, cfg config.Getter, gc git.Client, maxRecordsPerPool int, historyURI, statusURI string, logger *logrus.Entry) (*DefaultController, error) {
	if logger == nil {
		logger = logrus.NewEntry(logrus.StandardLogger())
	}
	hist, err := history.New(maxRecordsPerPool, historyURI)
	if err != nil {
		return nil, fmt.Errorf("error initializing history client from %q: %v", historyURI, err)
	}
	sc := &statusController{
		logger:         logger.WithField("controller", "status-update"),
		spc:            spcStatus,
		config:         cfg,
		newPoolPending: make(chan bool, 1),
		shutDown:       make(chan bool),
		path:           statusURI,
	}
	go sc.run()
	return &DefaultController{
		logger:         logger.WithField("controller", "sync"),
		spc:            spcSync,
		fileBrowsers:   fileBrowsers,
		launcherClient: launcherClient,
		tektonClient:   tektonClient,
		lhClient:       lighthouseClient,
		ns:             ns,
		config:         cfg,
		gc:             gc,
		sc:             sc,
		changedFiles: &changedFilesAgent{
			spc:             spcSync,
			nextChangeCache: make(map[changeCacheKey][]string),
		},
		History: hist,
	}, nil
}

// Shutdown signals the statusController to stop working and waits for it to
// finish its last update loop before terminating.
// DefaultController.Sync() should not be used after this function is called.
func (c *DefaultController) Shutdown() {
	err := c.gc.Clean()
	if err != nil {
		c.logger.Warnf("error cleaning local git cache: %s", err)
	}
	c.History.Flush()
	c.sc.shutdown()
}

// GetHistory returns the history
func (c *DefaultController) GetHistory() *history.History {
	return c.History
}

func (pr *PullRequest) prKey() string {
	return fmt.Sprintf("%s#%d", string(pr.Repository.NameWithOwner), int(pr.Number))
}

// newExpectedContext creates a Context with Expected state.
func newExpectedContext(c string) Context {
	return Context{
		Context:     githubql.String(c),
		State:       githubql.StatusStateExpected,
		Description: githubql.String(""),
	}
}

// contextsToStrings converts a list Context to a list of string
func contextsToStrings(contexts []Context) []string {
	var names []string
	for _, c := range contexts {
		names = append(names, string(c.Context))
	}
	return names
}

// Sync runs one sync iteration.
func (c *DefaultController) Sync() error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		c.logger.WithField("duration", duration.String()).Info("Synced")
		keeperMetrics.syncDuration.Set(duration.Seconds())
	}()
	defer c.changedFiles.prune()

	c.logger.Debug("Building keeper pool.")
	prs := make(map[string]PullRequest)
	if c.spc.SupportsGraphQL() {
		for _, query := range c.config().Keeper.Queries {
			q := query.Query()
			results, err := graphQLSearch(c.spc.Query, c.logger, q, time.Time{}, time.Now())
			if err != nil && len(results) == 0 {
				return fmt.Errorf("query %q, err: %v", q, err)
			}
			if err != nil {
				c.logger.WithError(err).WithField("query", q).Warning("found partial results")
			}

			for _, pr := range results {
				p := pr
				prs[p.prKey()] = pr
			}
		}
	} else {
		results, err := restAPISearch(c.spc, c.logger, c.config().Keeper.Queries, time.Time{}, time.Now())
		if err != nil {
			c.logger.WithError(err).Warnf("failed to perform REST query for PRs")
			return errors.Wrapf(err, "failed to perform REST query for PRs")
		}

		for _, pr := range results {
			p := pr
			prs[p.prKey()] = pr
		}
	}
	c.logger.WithField(
		"duration", time.Since(start).String(),
	).Debugf("Found %d (unfiltered) pool PRs.", len(prs))

	var lhjs []v1alpha1.LighthouseJob
	var blocks blockers.Blockers
	var err error
	if len(prs) > 0 {
		start := time.Now()
		lhjList, err := c.lhClient.LighthouseV1alpha1().LighthouseJobs(c.ns).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			c.logger.WithField("duration", time.Since(start).String()).Debug("Failed to list LighthouseJobs from the cluster.")
			return err
		}

		if len(lhjList.Items) > 200 {
			c.logger.Warn("Over 200+ lighthouse jobs in the cluster, this could lead to keeper failing readiness and liveness probes")
		}

		c.logger.WithField("duration", time.Since(start).String()).WithField("lighthouse-job-quantity", len(lhjList.Items)).Debug("Listed LighthouseJobs from the cluster.")
		lhjs = lhjList.Items

		// TODO: Support blockers with non-graphql
		if c.spc.SupportsGraphQL() {
			if label := c.config().Keeper.BlockerLabel; label != "" {
				c.logger.Debugf("Searching for blocking issues (label %q).", label)
				orgExcepts, repos := c.config().Keeper.Queries.OrgExceptionsAndRepos()
				orgs := make([]string, 0, len(orgExcepts))
				for org := range orgExcepts {
					orgs = append(orgs, org)
				}
				orgRepoQuery := orgRepoQueryString(orgs, repos.UnsortedList(), orgExcepts)
				blocks, err = blockers.FindAll(c.spc, c.logger, label, orgRepoQuery)
				if err != nil {
					return err
				}
			}
		}
	}
	// Partition PRs into subpools and filter out non-pool PRs.
	rawPools, err := c.dividePool(prs, lhjs)
	if err != nil {
		return err
	}
	filteredPools := c.filterSubpools(c.config().Keeper.MaxGoroutines, rawPools)

	// Sync subpools in parallel.
	poolChan := make(chan Pool, len(filteredPools))
	subpoolsInParallel(
		c.config().Keeper.MaxGoroutines,
		filteredPools,
		func(sp *subpool) {
			pool, err := c.syncSubpool(*sp, blocks.GetApplicable(sp.org, sp.repo, sp.branch))
			if err != nil {
				sp.log.WithError(err).Errorf("Error syncing subpool.")
			}
			poolChan <- pool
		},
	)

	close(poolChan)
	pools := make([]Pool, 0, len(poolChan))
	for pool := range poolChan {
		pools = append(pools, pool)
	}
	sortPools(pools)
	c.m.Lock()
	c.pools = pools
	// Notify statusController about the new pool.
	c.sc.Lock()
	c.sc.blocks = blocks
	c.sc.poolPRs = poolsToStatusPRMap(pools)
	select {
	case c.sc.newPoolPending <- true:
	default:
	}
	c.sc.Unlock()
	c.m.Unlock()

	c.History.Flush()
	return nil
}

func poolsToStatusPRMap(pools []Pool) map[string]prWithStatus {
	result := make(map[string]prWithStatus)

	for _, p := range pools {
		for k, v := range p.toPRsWithStatus() {
			result[k] = v
		}
	}

	return result
}

func (p *Pool) toPRsWithStatus() map[string]prWithStatus {
	result := make(map[string]prWithStatus)

	waitingFor := make(map[int]bool)
	waitingForBatch := make(map[int]bool)

	targets := make(map[int]bool)

	if p.Action == Merge || p.Action == MergeBatch {
		for _, t := range p.Target {
			targets[int(t.Number)] = true
		}
	}

	for _, w := range p.PendingPRs {
		waitingFor[int(w.Number)] = true
		result[w.prKey()] = prWithStatus{
			pr:      w,
			success: false,
		}
	}

	for _, b := range p.BatchPending {
		waitingForBatch[int(b.Number)] = true
		result[b.prKey()] = prWithStatus{
			pr:      b,
			success: false,
		}
	}

	for _, m := range p.MissingPRs {
		result[m.prKey()] = prWithStatus{
			pr:      m,
			success: false,
		}
	}

	for _, s := range p.SuccessPRs {
		out := prWithStatus{
			pr:              s,
			success:         true,
			waitingFor:      []int{},
			waitingForBatch: []int{},
			blocks:          p.Blockers,
		}
		// Add waiting for information for succeeded PRs
		_, inTargets := targets[int(s.Number)]
		_, inPending := waitingFor[int(s.Number)]
		_, inBatch := waitingFor[int(s.Number)]

		if !inTargets && !inPending && !inBatch {
			for k := range waitingFor {
				out.waitingFor = append(out.waitingFor, k)
			}
			for k := range waitingForBatch {
				out.waitingForBatch = append(out.waitingForBatch, k)
			}
		}
		result[s.prKey()] = out
	}

	return result
}

func (c *DefaultController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.m.Lock()
	defer c.m.Unlock()
	b, err := json.Marshal(c.pools)
	if err != nil {
		c.logger.WithError(err).Error("Encoding JSON.")
		b = []byte("[]")
	}
	if _, err = w.Write(b); err != nil {
		c.logger.WithError(err).Error("Writing JSON response.")
	}
}

// GetPools returns the pool status
func (c *DefaultController) GetPools() []Pool {
	c.m.Lock()
	defer c.m.Unlock()
	answer := []Pool{}
	answer = append(answer, c.pools...)
	return answer
}

func subpoolsInParallel(goroutines int, sps map[string]*subpool, process func(*subpool)) {
	// Load the subpools into a channel for use as a work queue.
	queue := make(chan *subpool, len(sps))
	for _, sp := range sps {
		queue <- sp
	}
	close(queue)

	if goroutines > len(queue) {
		goroutines = len(queue)
	}
	wg := &sync.WaitGroup{}
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for sp := range queue {
				process(sp)
			}
		}()
	}
	wg.Wait()
}

// filterSubpools filters non-pool PRs out of the initially identified subpools,
// deleting any pools that become empty.
// See filterSubpool for filtering details.
func (c *DefaultController) filterSubpools(goroutines int, raw map[string]*subpool) map[string]*subpool {
	filtered := make(map[string]*subpool)
	var lock sync.Mutex

	subpoolsInParallel(
		goroutines,
		raw,
		func(sp *subpool) {
			if err := c.initSubpoolData(sp); err != nil {
				sp.log.WithError(err).Error("Error initializing subpool.")
				return
			}
			key := poolKey(sp.org, sp.repo, sp.branch)
			if spFiltered := filterSubpool(c.spc, sp); spFiltered != nil {
				sp.log.WithField("key", key).WithField("pool", spFiltered).Debug("filtered sub-pool")

				lock.Lock()
				filtered[key] = spFiltered
				lock.Unlock()
			} else {
				sp.log.WithField("key", key).WithField("pool", spFiltered).Debug("filtering sub-pool removed all PRs")
			}
		},
	)
	return filtered
}

func (c *DefaultController) initSubpoolData(sp *subpool) error {
	var err error
	sp.presubmits, err = c.presubmitsByPull(sp)
	if err != nil {
		return fmt.Errorf("error determining required presubmit PipelineActivitys: %v", err)
	}
	sp.cc, err = c.config().GetKeeperContextPolicy(sp.org, sp.repo, sp.branch)
	if err != nil {
		return fmt.Errorf("error setting up context checker: %v", err)
	}
	return nil
}

// filterSubpool filters PRs from an initially identified subpool, returning the
// filtered subpool.
// If the subpool becomes empty 'nil' is returned to indicate that the subpool
// should be deleted.
func filterSubpool(spc scmProviderClient, sp *subpool) *subpool {
	var toKeep []PullRequest
	for _, pr := range sp.prs {
		p := pr
		if !filterPR(spc, sp, &p) {
			toKeep = append(toKeep, pr)
		}
	}
	if len(toKeep) == 0 {
		return nil
	}
	sp.prs = toKeep
	return sp
}

// filterPR indicates if a PR should be filtered out of the subpool.
// Specifically we filter out PRs that:
//   - Have known merge conflicts.
//   - Have failing or missing status contexts.
//   - Have pending required status contexts that are not associated with a
//     PipelineActivity. (This ensures that the 'keeper' context indicates that the pending
//     status is preventing merge. Required PipelineActivity statuses are allowed to be
//     'pending' because this prevents kicking PRs from the pool when Keeper is
//     retesting them.)
func filterPR(spc scmProviderClient, sp *subpool, pr *PullRequest) bool {
	log := sp.log.WithFields(pr.logFields())
	// Skip PRs that are known to be unmergeable.
	if pr.Mergeable == githubql.MergeableStateConflicting {
		log.Debug("filtering out PR as it is unmergeable")
		return true
	}
	// Filter out PRs with unsuccessful contexts unless the only unsuccessful
	// contexts are pending required PipelineActivitys.
	contexts, err := headContexts(log, spc, pr)
	if err != nil {
		log.WithError(err).Error("Getting head contexts.")
		return true
	}
	presubmitsHaveContext := func(context string) bool {
		for _, job := range sp.presubmits[int(pr.Number)] {
			if job.Context == context {
				return true
			}
		}
		return false
	}
	for _, ctx := range unsuccessfulContexts(contexts, sp.cc, log) {
		if ctx.State != githubql.StatusStatePending {
			log.WithField("context", ctx.Context).Debug("filtering out PR as unsuccessful context is not pending")
			return true
		}
		if !presubmitsHaveContext(string(ctx.Context)) {
			log.WithField("context", ctx.Context).Debug("filtering out PR as unsuccessful context is not Prow-controlled")
			return true
		}
	}

	return false
}

type simpleState string

const (
	failureState simpleState = "failure"
	pendingState simpleState = "pending"
	successState simpleState = "success"
)

func toSimpleState(s v1alpha1.PipelineState) simpleState {
	if s == v1alpha1.TriggeredState || s == v1alpha1.PendingState || s == v1alpha1.RunningState {
		return pendingState
	} else if s == v1alpha1.SuccessState {
		return successState
	}
	return failureState
}

// isPassingTests returns whether or not all contexts set on the PR except for
// the keeper pool context are passing.
func isPassingTests(log *logrus.Entry, spc scmProviderClient, pr PullRequest, cc contextChecker) bool {
	log = log.WithFields(pr.logFields())
	contexts, err := headContexts(log, spc, &pr)
	if err != nil {
		log.WithError(err).Error("Getting head commit status contexts.")
		// If we can't get the status of the commit, assume that it is failing.
		return false
	}
	unsuccessful := unsuccessfulContexts(contexts, cc, log)
	return len(unsuccessful) == 0
}

// unsuccessfulContexts determines which contexts from the list that we care about are
// failed. For instance, we do not care about our own context.
// If the branchProtection is set to only check for required checks, we will skip
// all non-required tests. If required tests are missing from the list, they will be
// added to the list of failed contexts.
func unsuccessfulContexts(contexts []Context, cc contextChecker, log *logrus.Entry) []Context {
	var failed []Context
	for _, ctx := range contexts {
		if string(ctx.Context) == GetStatusContextLabel() {
			continue
		}
		// Ignore legacy "tide" and "keeper" contexts
		if string(ctx.Context) == "keeper" || string(ctx.Context) == "tide" {
			continue
		}
		if cc.IsOptional(string(ctx.Context)) {
			continue
		}
		if ctx.State != githubql.StatusStateSuccess {
			failed = append(failed, ctx)
		}
	}
	for _, c := range cc.MissingRequiredContexts(contextsToStrings(contexts)) {
		failed = append(failed, newExpectedContext(c))
	}

	log.Debugf("from %d total contexts (%v) found %d failing contexts: %v", len(contexts), contextsToStrings(contexts), len(failed), contextsToStrings(failed))
	return failed
}

func pickSmallestPassingNumber(log *logrus.Entry, spc scmProviderClient, prs []PullRequest, cc contextChecker) (bool, PullRequest) {
	smallestNumber := -1
	var smallestPR PullRequest
	for _, pr := range prs {
		if smallestNumber != -1 && int(pr.Number) >= smallestNumber {
			continue
		}
		if len(pr.Commits.Nodes) < 1 {
			continue
		}
		if !isPassingTests(log, spc, pr, cc) {
			continue
		}
		smallestNumber = int(pr.Number)
		smallestPR = pr
	}
	return smallestNumber > -1, smallestPR
}

// accumulateBatch returns a list of PRs that can be merged after passing batch
// testing, if any exist. It also returns a list of PRs currently being batch
// tested.
func accumulateBatch(presubmits map[int][]job.Presubmit, prs []PullRequest, pjs []v1alpha1.LighthouseJob, log *logrus.Entry) ([]PullRequest, []PullRequest) {
	log.Debug("accumulating PRs for batch testing")
	if len(presubmits) == 0 {
		log.Debug("no presubmits configured, no batch can be triggered")
		return nil, nil
	}
	prNums := make(map[int]PullRequest)
	for _, pr := range prs {
		prNums[int(pr.Number)] = pr
	}
	type accState struct {
		prs       []PullRequest
		jobStates map[string]simpleState
		// Are the pull requests in the ref still acceptable? That is, do they
		// still point to the heads of the PRs?
		validPulls bool
	}
	states := make(map[string]*accState)
	for _, pj := range pjs {
		if pj.Spec.Type != job.BatchJob {
			continue
		}
		// First validate the batch job's refs.
		ref := pj.Spec.Refs.String()
		if _, ok := states[ref]; !ok {
			state := &accState{
				jobStates:  make(map[string]simpleState),
				validPulls: true,
			}
			for _, pull := range pj.Spec.Refs.Pulls {
				if pr, ok := prNums[pull.Number]; ok && string(pr.HeadRefOID) == pull.SHA {
					state.prs = append(state.prs, pr)
				} else if !ok {
					state.validPulls = false
					log.WithField("batch", ref).WithFields(pr.logFields()).Debug("batch job invalid, PR left pool")
					break
				} else {
					state.validPulls = false
					log.WithField("batch", ref).WithFields(pr.logFields()).Debug("batch job invalid, PR HEAD changed")
					break
				}
			}
			states[ref] = state
		}
		if !states[ref].validPulls {
			// The batch contains a PR ref that has changed. Skip it.
			continue
		}

		// Batch job refs are valid. Now accumulate job states by batch ref.
		context := pj.Spec.Context
		jobState := toSimpleState(pj.Status.State)

		// Store the best result for this ref+context.
		if s, ok := states[ref].jobStates[context]; !ok || s == failureState || jobState == successState {
			states[ref].jobStates[context] = jobState
		}
	}
	var pendingBatch, successBatch []PullRequest
	for ref, state := range states {
		if !state.validPulls {
			continue
		}
		requiredPresubmits := sets.NewString()
		for _, pr := range state.prs {
			for _, job := range presubmits[int(pr.Number)] {
				requiredPresubmits.Insert(job.Context)
			}
		}
		overallState := successState
		for _, p := range requiredPresubmits.List() {
			if s, ok := state.jobStates[p]; !ok || s == failureState {
				overallState = failureState
				log.WithField("batch", ref).Debugf("batch invalid, required presubmit %s is not passing", p)
				break
			} else if s == pendingState && overallState == successState {
				overallState = pendingState
			}
		}
		switch overallState {
		// Currently we only consider 1 pending batch and 1 success batch at a time.
		// If more are somehow present they will be ignored.
		case pendingState:
			pendingBatch = state.prs
		case successState:
			successBatch = state.prs
		}
	}
	return successBatch, pendingBatch
}

// accumulate returns the supplied PRs sorted into three buckets based on their
// accumulated state across the presubmits.
func accumulate(presubmits map[int][]job.Presubmit, prs []PullRequest, pjs []v1alpha1.LighthouseJob, log *logrus.Entry) (successes, pendings, missings []PullRequest, missingTests map[int][]job.Presubmit) {
	missingTests = map[int][]job.Presubmit{}
	for _, pr := range prs {
		// Accumulate the best result for each job (Passing > Pending > Failing/Unknown)
		// We can ignore the baseSHA here because the subPool only contains PipelineActivitys with the correct baseSHA
		psStates := make(map[string]simpleState)
		for _, pj := range pjs {
			if pj.Spec.Type != job.PresubmitJob {
				continue
			}
			if len(pj.Spec.Refs.Pulls) == 0 || pj.Spec.Refs.Pulls[0].Number != int(pr.Number) {
				continue
			}
			if pj.Spec.Refs.Pulls[0].SHA != string(pr.HeadRefOID) {
				continue
			}

			name := pj.Spec.Context
			oldState := psStates[name]
			newState := toSimpleState(pj.Status.State)
			if oldState == failureState || oldState == "" {
				psStates[name] = newState
			} else if oldState == pendingState && newState == successState {
				psStates[name] = successState
			}
		}
		// The overall result for the PR is the worst of the best of all its
		// required Presubmits
		overallState := successState
		for _, ps := range presubmits[int(pr.Number)] {
			if s, ok := psStates[ps.Context]; !ok {
				// No PJ with correct baseSHA+headSHA exists
				missingTests[int(pr.Number)] = append(missingTests[int(pr.Number)], ps)
				log.WithFields(pr.logFields()).Debugf("missing presubmit %s", ps.Context)
			} else if s == failureState {
				// PJ with correct baseSHA+headSHA exists but failed
				missingTests[int(pr.Number)] = append(missingTests[int(pr.Number)], ps)
				log.WithFields(pr.logFields()).Debugf("presubmit %s not passing", ps.Context)
			} else if s == pendingState {
				log.WithFields(pr.logFields()).Debugf("presubmit %s pending", ps.Context)
				overallState = pendingState
			}
		}
		if len(missingTests[int(pr.Number)]) > 0 {
			overallState = failureState
		}

		if overallState == successState {
			successes = append(successes, pr)
		} else if overallState == pendingState {
			pendings = append(pendings, pr)
		} else {
			missings = append(missings, pr)
		}
	}
	return
}

func prNumbers(prs []PullRequest) []int {
	var nums []int
	for _, pr := range prs {
		nums = append(nums, int(pr.Number))
	}
	return nums
}

func (c *DefaultController) pickBatch(sp subpool, cc contextChecker) ([]PullRequest, error) {
	batchLimit := c.config().Keeper.BatchSizeLimit(sp.org, sp.repo)
	if batchLimit < 0 {
		sp.log.Debug("Batch merges disabled by configuration in this repo.")
		return nil, nil
	}
	// we must choose the oldest PRs for the batch
	sort.Slice(sp.prs, func(i, j int) bool { return sp.prs[i].Number < sp.prs[j].Number })

	var candidates []PullRequest
	for _, pr := range sp.prs {
		if isPassingTests(sp.log, c.spc, pr, cc) {
			candidates = append(candidates, pr)
		}
	}

	if len(candidates) == 0 {
		sp.log.Debugf("of %d possible PRs, none were passing tests, no batch will be created", len(sp.prs))
		return nil, nil
	}
	sp.log.Debugf("of %d possible PRs, %d are passing tests", len(sp.prs), len(candidates))

	r, err := c.gc.Clone(sp.org + "/" + sp.repo)
	if err != nil {
		return nil, err
	}
	defer r.Clean() //nolint: errcheck
	if err := r.Config("user.name", "prow"); err != nil {
		return nil, err
	}
	if err := r.Config("user.email", "prow@localhost"); err != nil {
		return nil, err
	}
	if err := r.Config("commit.gpgsign", "false"); err != nil {
		sp.log.Warningf("Cannot set gpgsign=false in gitconfig: %v", err)
	}
	if err := r.Checkout(sp.sha); err != nil {
		return nil, err
	}

	var res []PullRequest
	for _, pr := range candidates {
		if ok, err := r.Merge(string(pr.HeadRefOID)); err != nil {
			// we failed to abort the merge and our git client is
			// in a bad state; it must be cleaned before we try again
			return nil, err
		} else if ok {
			res = append(res, pr)
			// TODO: Make this configurable per subpool.
			if batchLimit > 0 && len(res) >= batchLimit {
				break
			}
		}
	}
	return res, nil
}

func checkMergeLabels(pr PullRequest, squash, rebase, merge string, method keeper.PullRequestMergeType) (keeper.PullRequestMergeType, error) {
	labelCount := 0
	for _, prlabel := range pr.Labels.Nodes {
		switch string(prlabel.Name) {
		case squash:
			method = keeper.MergeSquash
			labelCount++
		case rebase:
			method = keeper.MergeRebase
			labelCount++
		case merge:
			method = keeper.MergeMerge
			labelCount++
		}
		if labelCount > 1 {
			return "", fmt.Errorf("conflicting merge method override labels")
		}
	}
	return method, nil
}

func (c *DefaultController) prepareMergeDetails(commitTemplates keeper.MergeCommitTemplate, pr PullRequest, mergeMethod keeper.PullRequestMergeType) scmprovider.MergeDetails {
	ghMergeDetails := scmprovider.MergeDetails{
		SHA:         string(pr.HeadRefOID),
		MergeMethod: string(mergeMethod),
	}

	if commitTemplates.Title != nil {
		var b bytes.Buffer

		if err := commitTemplates.Title.Execute(&b, pr); err != nil {
			c.logger.Errorf("error executing commit title template: %v", err)
		} else {
			ghMergeDetails.CommitTitle = b.String()
		}
	}

	if commitTemplates.Body != nil {
		var b bytes.Buffer

		if err := commitTemplates.Body.Execute(&b, pr); err != nil {
			c.logger.Errorf("error executing commit body template: %v", err)
		} else {
			ghMergeDetails.CommitMessage = b.String()
		}
	}

	return ghMergeDetails
}

func (c *DefaultController) mergePRs(sp subpool, prs []PullRequest) error {
	var merged, failed []int
	var failedPRs []PullRequest
	defer func() {
		if len(merged) == 0 {
			return
		}
		keeperMetrics.merges.WithLabelValues(sp.org, sp.repo, sp.branch).Observe(float64(len(merged)))
	}()

	var errs []error
	log := sp.log.WithField("merge-targets", prNumbers(prs))
	for i, pr := range prs {
		log := log.WithFields(pr.logFields())
		mergeMethod := c.config().Keeper.MergeMethod(sp.org, sp.repo)
		commitTemplates := c.config().Keeper.MergeCommitTemplate(sp.org, sp.repo)
		squashLabel := c.config().Keeper.SquashLabel
		rebaseLabel := c.config().Keeper.RebaseLabel
		mergeLabel := c.config().Keeper.MergeLabel
		if squashLabel != "" || rebaseLabel != "" || mergeLabel != "" {
			var err error
			mergeMethod, err = checkMergeLabels(pr, squashLabel, rebaseLabel, mergeLabel, mergeMethod)
			if err != nil {
				log.WithError(err).Error("Merge failed.")
				errs = append(errs, err)
				failed = append(failed, int(pr.Number))
				failedPRs = append(failedPRs, pr)
				continue
			}
		}

		keepTrying, err := tryMerge(func() error {
			ghMergeDetails := c.prepareMergeDetails(commitTemplates, pr, mergeMethod)
			return c.spc.Merge(sp.org, sp.repo, int(pr.Number), ghMergeDetails)
		})
		if err != nil {
			log.WithError(err).Error("Merge failed.")
			errs = append(errs, err)
			failed = append(failed, int(pr.Number))
			failedPRs = append(failedPRs, pr)
		} else {
			log.Info("Merged.")
			merged = append(merged, int(pr.Number))
		}
		if !keepTrying {
			break
		}
		// If we successfully merged this PR and have more to merge, sleep to give
		// GitHub time to recalculate mergeability.
		if err == nil && i+1 < len(prs) {
			sleep(time.Second * 5)
		}
	}

	if len(errs) == 0 {
		return nil
	}

	finalErr := rollupMergeErrors(prs, failed, merged, errs)
	reportErr := c.commentOnPRsWithFailedMerge(failedPRs, finalErr.Error())
	if reportErr != nil {
		sp.log.WithFields(logrus.Fields{
			"targets": prNumbers(failedPRs),
		}).WithError(reportErr).Error("error reporting back to PRs upon failed merge")
	}

	return finalErr
}

func rollupMergeErrors(prs []PullRequest, failed []int, merged []int, errs []error) error {
	// Construct a more informative error.
	var batch string
	if len(prs) > 1 {
		batch = fmt.Sprintf(" from batch %v", prNumbers(prs))
		if len(merged) > 0 {
			batch = fmt.Sprintf("%s, partial merge %v", batch, merged)
		}
	}

	return fmt.Errorf("failed merging %v%s: %v", failed, batch, errorutil.NewAggregate(errs...))
}

func mergeErrorDetail(origErr error) error {
	switch origErr.(type) {
	case scmprovider.ModifiedHeadError:
		return fmt.Errorf("PR was modified: %v", origErr)
	case scmprovider.UnmergablePRBaseChangedError:
		return fmt.Errorf("base branch was modified: %v", origErr)
	case scmprovider.UnauthorizedToPushError:
		return fmt.Errorf("branch needs to be configured to allow this robot to push: %v", origErr)
	case scmprovider.MergeCommitsForbiddenError:
		return fmt.Errorf("keeper needs to be configured to use the 'rebase' merge method for this repo or the repo needs to allow merge commits: %v", origErr)
	case scmprovider.UnmergablePRError:
		return fmt.Errorf("PR is unmergable. Do the Keeper merge requirements match the SCM provider settings for the repo? %v", origErr)
	default:
		return origErr
	}
}

// tryMerge attempts 1 merge and returns a bool indicating if we should try
// to merge the remaining PRs and possibly an error.
func tryMerge(mergeFunc func() error) (bool, error) {
	var err error
	const maxRetries = 3
	backoff := time.Second * 4
	for retry := 0; retry < maxRetries; retry++ {
		if err = mergeFunc(); err == nil {
			// Successful merge!
			return true, nil
		}
		detailedErr := mergeErrorDetail(err)
		// TODO: Add a config option to abort batches if a PR in the batch
		// cannot be merged for any reason. This would skip merging
		// not just the changed PR, but also the other PRs in the batch.
		// This shouldn't be the default behavior as merging batches is high
		// priority and this is unlikely to be problematic.
		// Note: We would also need to be able to roll back any merges for the
		// batch that were already successfully completed before the failure.
		// Ref: https://github.com/kubernetes/test-infra/issues/10621
		if _, ok := err.(scmprovider.ModifiedHeadError); ok {
			// This is a possible source of incorrect behavior. If someone
			// modifies their PR as we try to merge it in a batch then we
			// end up in an untested state. This is unlikely to cause any
			// real problems.
			return true, detailedErr
		} else if _, ok = err.(scmprovider.UnmergablePRBaseChangedError); ok {
			//  complained that the base branch was modified. This is a
			// strange error because the API doesn't even allow the request to
			// specify the base branch sha, only the head sha.
			// We suspect that github is complaining because we are making the
			// merge requests too rapidly and it cannot recompute mergability
			// in time. https://gitprovider.com/kubernetes/test-infra/issues/5171
			// We handle this by sleeping for a few seconds before trying to
			// merge again.
			err = detailedErr
			if retry+1 < maxRetries {
				sleep(backoff)
				backoff *= 2
			}
		} else if _, ok = err.(scmprovider.UnauthorizedToPushError); ok {
			// GitHub let us know that the token used cannot push to the branch.
			// Even if the robot is set up to have write access to the repo, an
			// overzealous branch protection setting will not allow the robot to
			// push to a specific branch.
			// We won't be able to merge the other PRs.
			return false, detailedErr
		} else if _, ok = err.(scmprovider.MergeCommitsForbiddenError); ok {
			// GitHub let us know that the merge method configured for this repo
			// is not allowed by other repo settings, so we should let the admins
			// know that the configuration needs to be updated.
			// We won't be able to merge the other PRs.
			return false, detailedErr
		} else if _, ok = err.(scmprovider.UnmergablePRError); ok {
			return true, detailedErr
		} else {
			return true, err
		}
	}
	// We ran out of retries. Return the last transient error.
	return true, err
}

func (c *DefaultController) trigger(sp subpool, presubmits map[int][]job.Presubmit, prs []PullRequest) error {
	refs := v1alpha1.Refs{
		Org:      sp.org,
		Repo:     sp.repo,
		BaseRef:  sp.branch,
		BaseSHA:  sp.sha,
		CloneURI: sp.cloneURL,
	}
	for _, pr := range prs {
		refs.Pulls = append(
			refs.Pulls,
			v1alpha1.Pull{
				Number: int(pr.Number),
				Author: string(pr.Author.Login),
				SHA:    string(pr.HeadRefOID),
				Ref:    fmt.Sprintf(c.spc.PRRefFmt(), int(pr.Number)),
			},
		)
	}

	// If PRs require the same job, we only want to trigger it once.
	// If multiple required jobs have the same context, we assume the
	// same shard will be run to provide those contexts
	triggeredContexts := sets.NewString()
	for _, pr := range prs {
		for _, ps := range presubmits[int(pr.Number)] {
			if triggeredContexts.Has(ps.Context) {
				continue
			}
			triggeredContexts.Insert(ps.Context)
			var spec v1alpha1.LighthouseJobSpec
			if len(prs) == 1 {
				spec = jobutil.PresubmitSpec(c.logger, ps, refs)
			} else {
				spec = jobutil.BatchSpec(c.logger, ps, refs)
			}
			pj := jobutil.NewLighthouseJob(spec, ps.Labels, ps.Annotations)
			start := time.Now()
			c.logger.WithFields(jobutil.LighthouseJobFields(&pj)).Info("Creating a new LighthouseJob.")
			if _, err := c.launcherClient.Launch(&pj); err != nil {
				c.logger.WithField("duration", time.Since(start).String()).Debug("Failed to create pipeline on the cluster.")
				return fmt.Errorf("failed to create a pipeline for job: %q, PRs: %v: %v", spec.Job, prNumbers(prs), err)
			}
			sha := refs.BaseSHA
			if len(refs.Pulls) > 0 {
				sha = refs.Pulls[0].SHA
			}

			statusInput := &scm.StatusInput{
				State: scm.StatePending,
				Label: spec.Context,
				Desc:  util.CommitStatusPendingDescription,
			}
			if _, err := c.spc.CreateStatus(refs.Org, refs.Repo, sha, statusInput); err != nil {
				c.logger.WithField("duration", time.Since(start).String()).Debug("Failed to set pending status on triggered context.")
				return errors.Wrapf(err, "Cannot update PR status on org %s repo %s sha %s for context %s", refs.Org, refs.Repo, sha, statusInput.Label)
			}
			c.logger.WithField("duration", time.Since(start).String()).Debug("Created pipeline on the cluster.")
		}
	}
	return nil
}

func (c *DefaultController) takeAction(sp subpool, batchPending, successes, pendings, missings, batchMerges []PullRequest, missingSerialTests map[int][]job.Presubmit) (Action, []PullRequest, error) {
	// Merge the batch!
	if len(batchMerges) > 0 {
		return MergeBatch, batchMerges, c.mergePRs(sp, batchMerges)
	}
	// Do not merge PRs while waiting for a batch to complete. We don't want to
	// invalidate the old batch result.
	if len(successes) > 0 && len(batchPending) == 0 {
		if ok, pr := pickSmallestPassingNumber(sp.log, c.spc, successes, sp.cc); ok {
			return Merge, []PullRequest{pr}, c.mergePRs(sp, []PullRequest{pr})
		}
	}
	// If no presubmits are configured, just wait.
	if len(sp.presubmits) == 0 {
		return Wait, nil, nil
	}
	// If we have no batch, trigger one.
	if len(sp.prs) > 1 && len(batchPending) == 0 {
		batch, err := c.pickBatch(sp, sp.cc)
		if err != nil {
			return Wait, nil, err
		}
		if len(batch) > 1 {
			sp.log.Infof("triggering batch job")
			return TriggerBatch, batch, c.trigger(sp, sp.presubmits, batch)
		}
	}
	disableTrigger := strings.ToLower(os.Getenv("LIGHTHOUSE_TRIGGER_ON_MISSING"))
	if disableTrigger == "disable" || strings.HasPrefix(disableTrigger, "disable") {
		return Wait, nil, nil
	}
	if disableTrigger != "" {
		fullName := sp.org + "/" + sp.repo
		disableRepos := strings.Split(disableTrigger, ",")
		for _, disableRepo := range disableRepos {
			if strings.TrimSpace(disableRepo) == fullName {
				return Wait, nil, nil
			}
		}
	}
	// If we have no serial jobs pending or successful, trigger one.
	if len(missings) > 0 && len(pendings) == 0 && len(successes) == 0 {
		if ok, pr := pickSmallestPassingNumber(sp.log, c.spc, missings, sp.cc); ok {
			sp.log.Infof("triggering job as we have missings %d and no pendings and no successes", len(missings))
			return Trigger, []PullRequest{pr}, c.trigger(sp, missingSerialTests, []PullRequest{pr})
		}
	}
	return Wait, nil, nil
}

// changedFilesAgent queries and caches the names of files changed by PRs.
// Cache entries expire if they are not used during a sync loop.
type changedFilesAgent struct {
	spc         scmProviderClient
	changeCache map[changeCacheKey][]string
	// nextChangeCache caches file change info that is relevant this sync for use next sync.
	// This becomes the new changeCache when prune() is called at the end of each sync.
	nextChangeCache map[changeCacheKey][]string
	sync.RWMutex
}

type changeCacheKey struct {
	org, repo string
	number    int
	sha       string
}

// prChanges gets the files changed by the PR, either from the cache or by
// querying GitHub.
func (c *changedFilesAgent) prChanges(pr *PullRequest) job.ChangedFilesProvider {
	return func() ([]string, error) {
		cacheKey := changeCacheKey{
			org:    string(pr.Repository.Owner.Login),
			repo:   string(pr.Repository.Name),
			number: int(pr.Number),
			sha:    string(pr.HeadRefOID),
		}

		c.RLock()
		changedFiles, ok := c.changeCache[cacheKey]
		if ok {
			c.RUnlock()
			c.Lock()
			c.nextChangeCache[cacheKey] = changedFiles
			c.Unlock()
			return changedFiles, nil
		}
		if changedFiles, ok = c.nextChangeCache[cacheKey]; ok {
			c.RUnlock()
			return changedFiles, nil
		}
		c.RUnlock()

		// We need to query the changes from GitHub.
		changes, err := c.spc.GetPullRequestChanges(
			string(pr.Repository.Owner.Login),
			string(pr.Repository.Name),
			int(pr.Number),
		)
		if err != nil {
			return nil, fmt.Errorf("error getting PR changes for #%d: %v", int(pr.Number), err)
		}
		changedFiles = make([]string, 0, len(changes))
		for _, change := range changes {
			changedFiles = append(changedFiles, change.Path)
		}

		c.Lock()
		c.nextChangeCache[cacheKey] = changedFiles
		c.Unlock()
		return changedFiles, nil
	}
}

// prune removes any cached file changes that were not used since the last prune.
func (c *changedFilesAgent) prune() {
	c.Lock()
	defer c.Unlock()
	c.changeCache = c.nextChangeCache
	c.nextChangeCache = make(map[changeCacheKey][]string)
}

func (c *DefaultController) presubmitsByPull(sp *subpool) (map[int][]job.Presubmit, error) {
	presubmits := make(map[int][]job.Presubmit, len(sp.prs))
	record := func(num int, j job.Presubmit) {
		if jobs, ok := presubmits[num]; ok {
			presubmits[num] = append(jobs, j)
		} else {
			presubmits[num] = []job.Presubmit{j}
		}
	}

	// lets get the in repo config for the repo
	owner := sp.org
	repo := sp.repo
	sharedConfig := c.config()
	cache := inrepo.NewResolverCache()
	fc := filebrowser.NewFetchCache()
	cfg, _, err := inrepo.Generate(c.fileBrowsers, fc, cache, sharedConfig, nil, owner, repo, "")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to calculate in repo config")
	}

	for _, ps := range cfg.Presubmits[owner+"/"+repo] {
		if !ps.ContextRequired() {
			continue
		}

		for _, pr := range sp.prs {
			p := pr
			if shouldRun, err := ps.ShouldRun(sp.branch, c.changedFiles.prChanges(&p), false, false); err != nil {
				return nil, err
			} else if shouldRun {
				record(int(pr.Number), ps)
			}
		}
	}
	return presubmits, nil
}

func (c *DefaultController) commentOnPRsWithFailedMerge(prs []PullRequest, errorString string) error {
	timestamp := time.Now().Format(time.RFC1123) // Format the time to a readable string
	commentBody := fmt.Sprintf("Failed to merge this PR at %s due to:\n>%s\n", timestamp, errorString)

	var errs []error
	for _, pr := range prs {
		// Fetch existing comments
		comments, err := c.spc.ListPullRequestComments(string(pr.Repository.Owner.Login), string(pr.Repository.Name), int(pr.Number))
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// order comments by creation date
		sort.Slice(comments, func(i, j int) bool {
			return comments[i].Created.Before(comments[j].Created)
		})

		// Find the last keeper comment
		var lastKeeperCommentID *int
		// check the last comment
		for i := len(comments) - 1; i >= 0; i-- {
			if strings.Contains(comments[i].Body, "Failed to merge this PR at") {
				lastKeeperCommentID = &comments[i].ID
				break
			}
		}

		// Update the existing comment if found, otherwise create a new one
		if lastKeeperCommentID != nil {
			err = c.spc.EditComment(string(pr.Repository.Owner.Login), string(pr.Repository.Name), *lastKeeperCommentID, int(pr.Number), commentBody, true)
		} else {
			err = c.spc.CreateComment(string(pr.Repository.Owner.Login), string(pr.Repository.Name), int(pr.Number), true, commentBody)
		}

		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Wrap(errorutil.NewAggregate(errs...), "error reporting on failed merges")
	}
	return nil
}

func (c *DefaultController) syncSubpool(sp subpool, blocks []blockers.Blocker) (Pool, error) {
	sp.log.Infof("Syncing subpool: %d PRs, %d LJs.", len(sp.prs), len(sp.ljs))
	successes, pendings, missings, missingSerialTests := accumulate(sp.presubmits, sp.prs, sp.ljs, sp.log)
	batchMerge, batchPending := accumulateBatch(sp.presubmits, sp.prs, sp.ljs, sp.log)
	sp.log.WithFields(logrus.Fields{
		"prs-passing":   prNumbers(successes),
		"prs-pending":   prNumbers(pendings),
		"prs-missing":   prNumbers(missings),
		"batch-passing": prNumbers(batchMerge),
		"batch-pending": prNumbers(batchPending),
	}).Info("Subpool accumulated.")

	var act Action
	var targets []PullRequest
	var err error
	var errorString string
	if len(blocks) > 0 {
		act = PoolBlocked
	} else {
		act, targets, err = c.takeAction(sp, batchPending, successes, pendings, missings, batchMerge, missingSerialTests)
		if err != nil {
			errorString = err.Error()
		}
		if recordableActions[act] {
			c.History.Record(
				poolKey(sp.org, sp.repo, sp.branch),
				string(act),
				sp.sha,
				errorString,
				prMeta(targets...),
			)
		}
	}

	sp.log.WithFields(logrus.Fields{
		"action":  string(act),
		"targets": prNumbers(targets),
	}).Info("Subpool synced.")
	keeperMetrics.pooledPRs.WithLabelValues(sp.org, sp.repo, sp.branch).Set(float64(len(sp.prs)))
	keeperMetrics.updateTime.WithLabelValues(sp.org, sp.repo, sp.branch).Set(float64(time.Now().Unix()))
	return Pool{
			Org:    sp.org,
			Repo:   sp.repo,
			Branch: sp.branch,

			SuccessPRs: successes,
			PendingPRs: pendings,
			MissingPRs: missings,

			BatchPending: batchPending,

			Action:   act,
			Target:   targets,
			Blockers: blocks,
			Error:    errorString,
		},
		err
}

func prMeta(prs ...PullRequest) []v1alpha1.Pull {
	var res []v1alpha1.Pull
	for _, pr := range prs {
		res = append(res, v1alpha1.Pull{
			Number: int(pr.Number),
			Author: string(pr.Author.Login),
			Title:  string(pr.Title),
			SHA:    string(pr.HeadRefOID),
		})
	}
	return res
}

func sortPools(pools []Pool) {
	sort.Slice(pools, func(i, j int) bool {
		if string(pools[i].Org) != string(pools[j].Org) {
			return string(pools[i].Org) < string(pools[j].Org)
		}
		if string(pools[i].Repo) != string(pools[j].Repo) {
			return string(pools[i].Repo) < string(pools[j].Repo)
		}
		return string(pools[i].Branch) < string(pools[j].Branch)
	})

	sortPRs := func(prs []PullRequest) {
		sort.Slice(prs, func(i, j int) bool { return int(prs[i].Number) < int(prs[j].Number) })
	}
	for i := range pools {
		sortPRs(pools[i].SuccessPRs)
		sortPRs(pools[i].PendingPRs)
		sortPRs(pools[i].MissingPRs)
		sortPRs(pools[i].BatchPending)
	}
}

type subpool struct {
	log      *logrus.Entry
	org      string
	repo     string
	branch   string
	cloneURL string
	// sha is the baseSHA for this subpool
	sha string

	// ljs contains all LighthouseJobs of type Presubmit or Batch
	// that have the same baseSHA as the subpool
	ljs []v1alpha1.LighthouseJob
	prs []PullRequest

	cc contextChecker
	// presubmit contains all required presubmits for each PR
	// in this subpool
	presubmits map[int][]job.Presubmit
}

func poolKey(org, repo, branch string) string {
	return fmt.Sprintf("%s/%s:%s", org, repo, branch)
}

// dividePool splits up the list of pull requests and prow jobs into a group
// per repo and branch. It only keeps PipelineActivitys that match the latest branch.
func (c *DefaultController) dividePool(pool map[string]PullRequest, pjs []v1alpha1.LighthouseJob) (map[string]*subpool, error) {
	sps := make(map[string]*subpool)
	for _, pr := range pool {
		org := string(pr.Repository.Owner.Login)
		repo := string(pr.Repository.Name)
		branch := string(pr.BaseRef.Name)
		cloneURL := string(pr.Repository.URL)
		baseSHA := string(pr.BaseRef.Target.OID)
		if cloneURL == "" {
			return nil, errors.New("no clone URL specified for repository")
		}
		if !strings.HasSuffix(cloneURL, ".git") {
			cloneURL = cloneURL + ".git"
		}
		fn := poolKey(org, repo, branch)
		if sps[fn] == nil {
			sps[fn] = &subpool{
				log: c.logger.WithFields(logrus.Fields{
					"org":       org,
					"repo":      repo,
					"branch":    branch,
					"base-sha":  baseSHA,
					"clone-url": cloneURL,
				}),
				org:      org,
				repo:     repo,
				branch:   branch,
				sha:      baseSHA,
				cloneURL: cloneURL,
			}
		}
		sps[fn].prs = append(sps[fn].prs, pr)
	}
	for _, pj := range pjs {
		if pj.Spec.Type != job.PresubmitJob && pj.Spec.Type != job.BatchJob {
			continue
		}
		fn := poolKey(pj.Spec.Refs.Org, pj.Spec.Refs.Repo, pj.Spec.Refs.BaseRef)
		if sps[fn] == nil || pj.Spec.Refs.BaseSHA != sps[fn].sha {
			continue
		}
		sps[fn].ljs = append(sps[fn].ljs, pj)
	}
	return sps, nil
}

// GraphQLAuthor represents the author in the GitHub GraphQL layout
type GraphQLAuthor struct {
	Login githubql.String
}

// GraphQLBaseRef represents the author in the GitHub GraphQL layout
type GraphQLBaseRef struct {
	Name   githubql.String
	Prefix githubql.String
	Target struct {
		OID githubql.String `graphql:"oid"`
	}
}

// PullRequest holds graphql data about a PR, including its commits and their contexts.
type PullRequest struct {
	Number      githubql.Int
	Author      GraphQLAuthor
	BaseRef     GraphQLBaseRef
	HeadRefName githubql.String `graphql:"headRefName"`
	HeadRefOID  githubql.String `graphql:"headRefOid"`
	Mergeable   githubql.MergeableState
	Repository  Repository
	Commits     struct {
		Nodes []struct {
			Commit Commit
		}
		// Request the 'last' 4 commits hoping that one of them is the logically 'last'
		// commit with OID matching HeadRefOID. If we don't find it we have to use an
		// additional API token. (see the 'headContexts' func for details)
		// We can't raise this too much or we could hit the limit of 50,000 nodes
		// per query: https://developer.github.com/v4/guides/resource-limitations/#node-limit
	} `graphql:"commits(last: 4)"`
	Labels struct {
		Nodes []struct {
			Name githubql.String
		}
	} `graphql:"labels(first: 100)"`
	Milestone *struct {
		Title githubql.String
	}
	Body      githubql.String
	Title     githubql.String
	UpdatedAt githubql.DateTime
}

// Repository holds graphql/query data about repositories
type Repository struct {
	Name          githubql.String
	NameWithOwner githubql.String
	URL           githubql.String
	Owner         SCMUser
}

// SCMUser holds the username
type SCMUser struct {
	Login githubql.String
}

// Commit holds graphql data about commits and which contexts they have
type Commit struct {
	Status struct {
		Contexts []Context
	}
	OID githubql.String `graphql:"oid"`
}

// Context holds graphql response data for github contexts.
type Context struct {
	Context     githubql.String
	Description githubql.String
	State       githubql.StatusState
}

// PRNode a node containing a PR
type PRNode struct {
	PullRequest PullRequest `graphql:"... on PullRequest"`
}

type searchQuery struct {
	RateLimit struct {
		Cost      githubql.Int
		Remaining githubql.Int
	}
	Search struct {
		PageInfo struct {
			HasNextPage githubql.Boolean
			EndCursor   githubql.String
		}
		Nodes []PRNode
	} `graphql:"search(type: ISSUE, first: 100, after: $searchCursor, query: $query)"`
}

func (pr *PullRequest) logFields() logrus.Fields {
	return logrus.Fields{
		"org":  string(pr.Repository.Owner.Login),
		"repo": string(pr.Repository.Name),
		"pr":   int(pr.Number),
		"sha":  string(pr.HeadRefOID),
	}
}

// headContexts gets the status contexts for the commit with OID == pr.HeadRefOID
//
// First, we try to get this value from the commits we got with the PR query.
// Unfortunately the 'last' commit ordering is determined by author date
// not commit date so if commits are reordered non-chronologically on the PR
// branch the 'last' commit isn't necessarily the logically last commit.
// We list multiple commits with the query to increase our chance of success,
// but if we don't find the head commit we have to ask GitHub for it
// specifically (this costs an API token).
func headContexts(log *logrus.Entry, spc scmProviderClient, pr *PullRequest) ([]Context, error) {
	for _, node := range pr.Commits.Nodes {
		if node.Commit.OID == pr.HeadRefOID {
			return node.Commit.Status.Contexts, nil
		}
	}
	// We didn't get the head commit from the query (the commits must not be
	// logically ordered) so we need to specifically ask GitHub for the status
	// and coerce it to a graphql type.
	org := string(pr.Repository.Owner.Login)
	repo := string(pr.Repository.Name)
	// Log this event so we can tune the number of commits we list to minimize this.
	log.Warnf("'last' %d commits didn't contain logical last commit. Querying GitHub...", len(pr.Commits.Nodes))
	combined, err := spc.GetCombinedStatus(org, repo, string(pr.HeadRefOID))
	if err != nil {
		return nil, fmt.Errorf("failed to get the combined status: %v", err)
	}
	contexts := make([]Context, 0, len(combined.Statuses))
	for _, status := range combined.Statuses {
		contexts = append(
			contexts,
			Context{
				Context:     githubql.String(status.Label),
				Description: githubql.String(status.Desc),
				State:       githubql.StatusState(strings.ToUpper(status.State.String())),
			},
		)
	}
	// Add a commit with these contexts to pr for future look ups.
	pr.Commits.Nodes = append(pr.Commits.Nodes,
		struct{ Commit Commit }{
			Commit: Commit{
				OID:    pr.HeadRefOID,
				Status: struct{ Contexts []Context }{Contexts: contexts},
			},
		},
	)
	return contexts, nil
}

func orgRepoQueryString(orgs, repos []string, orgExceptions map[string]sets.String) string {
	toks := make([]string, 0, len(orgs))
	for _, o := range orgs {
		toks = append(toks, fmt.Sprintf("org:\"%s\"", o))

		for _, e := range orgExceptions[o].List() {
			toks = append(toks, fmt.Sprintf("-repo:\"%s\"", e))
		}
	}
	for _, r := range repos {
		toks = append(toks, fmt.Sprintf("repo:\"%s\"", r))
	}
	return strings.Join(toks, " ")
}

func reposToQueries(queries keeper.Queries) map[string][]keeper.Query {
	queryMap := make(map[string][]keeper.Query)
	// Create a map of each repo to the relevant queries
	for _, q := range queries {
		for _, repo := range q.Repos {
			queryMap[repo] = append(queryMap[repo], q)
		}
	}
	return queryMap
}

func restAPISearch(spc scmProviderClient, log *logrus.Entry, queries keeper.Queries, start, end time.Time) ([]PullRequest, error) {
	var relevantPRs []PullRequest

	queryMap := reposToQueries(queries)

	// Iterate over the repo list and query them
	for repo := range queryMap {
		searchOpts := scm.PullRequestListOptions{
			Page:   1,
			Size:   100,
			Open:   true,
			Closed: false,
		}
		if !start.Equal(time.Time{}) {
			searchOpts.UpdatedAfter = &start
		}
		if !end.Equal(time.Time{}) {
			searchOpts.UpdatedBefore = &end
		}

		prs, err := spc.ListAllPullRequestsForFullNameRepo(repo, searchOpts)
		if err != nil {
			log.WithError(err).Warnf("listing all open pull requests for %s failed, skipping repository", repo)
			continue
		}

		var repoData *scm.Repository

		// Iterate over the PRs to see if they match the relevant queries
		for _, pr := range prs {
			err = loadMissingLabels(spc, pr)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to load labels for PR %s", pr.Link)
			}
			prLabels := make(map[string]struct{})
			for _, l := range pr.Labels {
				prLabels[l.Name] = struct{}{}
			}
			matches := false
			for _, q := range queryMap[repo] {
				missingRequiredLabels := false
				for _, requiredLabel := range q.Labels {
					if _, ok := prLabels[requiredLabel]; !ok {
						// Required label not present, break
						missingRequiredLabels = true
						break
					}
				}

				hasExcludedLabel := false
				// Check if any of the excluded labels are present
				for _, excludedLabel := range q.MissingLabels {
					if _, ok := prLabels[excludedLabel]; ok {
						// Excluded label present, break
						hasExcludedLabel = true
						break
					}
				}

				hasExcludedBranch := false
				if len(q.ExcludedBranches) > 0 {
					for _, eb := range q.ExcludedBranches {
						if pr.Target == eb {
							hasExcludedBranch = true
							break
						}
					}
				}

				hasIncludedBranch := true
				if len(q.IncludedBranches) > 0 {
					hasIncludedBranch = false
					for _, ib := range q.IncludedBranches {
						if pr.Target == ib {
							hasIncludedBranch = true
							break
						}
					}
				}

				if !missingRequiredLabels && !hasExcludedLabel && !hasExcludedBranch && hasIncludedBranch {
					matches = true
					break
				}
			}

			if matches {
				// If this PR matches, get the repository details
				if repoData == nil {
					scmRepo, err := spc.GetRepositoryByFullName(repo)
					if err != nil {
						return nil, errors.Wrapf(err, "getting repository details for %s", repo)
					}
					repoData = scmRepo
				}

				gpr := scmPRToGraphQLPR(pr, repoData)
				_, err := headContexts(log, spc, gpr)
				if err != nil {
					log.WithError(err).Error("Getting head contexts but ignoring.")
				}

				relevantPRs = append(relevantPRs, *gpr)
			}
		}
	}

	return relevantPRs, nil
}

func loadMissingLabels(spc scmProviderClient, pr *scm.PullRequest) error {
	if len(pr.Labels) > 0 {
		return nil
	}
	gitKind := os.Getenv("GIT_KIND")
	if gitKind != "bitbucketserver" && gitKind != "bitbucketcloud" {
		return nil
	}

	// lets load the labels if they are missing
	repo := pr.Repository()
	var err error
	pr.Labels, err = spc.GetIssueLabels(repo.Namespace, repo.Name, pr.Number, true)
	if err != nil {
		return errors.Wrapf(err, "failed to find labels for PullRequest %s", pr.Link)
	}
	return nil
}

func scmPRToGraphQLPR(scmPR *scm.PullRequest, scmRepo *scm.Repository) *PullRequest {
	author := GraphQLAuthor{Login: githubql.String(scmPR.Author.Login)}

	baseRef := GraphQLBaseRef{
		Name:   githubql.String(scmPR.Target),
		Prefix: githubql.String(strings.TrimSuffix(scmPR.Base.Ref, scmPR.Target)),
		Target: struct {
			OID githubql.String `graphql:"oid"`
		}{OID: githubql.String(scmPR.Base.Sha)},
	}

	if baseRef.Prefix == "" {
		baseRef.Prefix = "refs/heads/"
	}

	mergeable := githubql.MergeableStateUnknown
	switch scmPR.MergeableState {
	case scm.MergeableStateMergeable:
		mergeable = githubql.MergeableStateMergeable
	case scm.MergeableStateConflicting:
		mergeable = githubql.MergeableStateConflicting
	}

	labels := struct {
		Nodes []struct {
			Name githubql.String
		}
	}{}
	for _, l := range scmPR.Labels {
		labels.Nodes = append(labels.Nodes, struct{ Name githubql.String }{Name: githubql.String(l.Name)})
	}

	return &PullRequest{
		Number:      githubql.Int(scmPR.Number),
		Author:      author,
		BaseRef:     baseRef,
		HeadRefName: githubql.String(scmPR.Source),
		HeadRefOID:  githubql.String(scmPR.Head.Sha),
		Mergeable:   mergeable,
		Repository:  scmRepoToGraphQLRepo(scmRepo),
		Labels:      labels,
		Body:        githubql.String(scmPR.Body),
		Title:       githubql.String(scmPR.Title),
		UpdatedAt:   githubql.DateTime{Time: scmPR.Updated},
	}
}

func scmRepoToGraphQLRepo(scmRepo *scm.Repository) Repository {
	return Repository{
		Name:          githubql.String(scmRepo.Name),
		NameWithOwner: githubql.String(scmRepo.FullName),
		URL:           githubql.String(scmRepo.Clone),
		Owner:         SCMUser{Login: githubql.String(scmRepo.Namespace)},
	}
}
