package tekton

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/jenkins-x/lighthouse/pkg/util"

	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/jobutil"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	controllerName          = "tekton-controller"
	gitCloneCatalogTaskName = "git-clone"
	gitCloneURLParam        = "url"
	gitCloneRevisionParam   = "revision"
	gitMergeCatalogTaskName = "git-batch-merge"
	gitMergeBatchRefsParam  = "batchedRefs"
)

type buildIDGenerator interface {
	GenerateBuildID() string
}

type epochBuildIDGenerator struct{}

// GenerateBuildID returns a string representation of milliseconds since epoch
func (e *epochBuildIDGenerator) GenerateBuildID() string {
	return strconv.FormatInt(time.Now().UnixNano()/1000000, 10)
}

var reProwExtraRef = regexp.MustCompile(`PROW_EXTRA_GIT_REF_(\d+)`)

func trimDashboardURL(base string) string {
	return strings.TrimSuffix(strings.TrimSuffix(base, "#"), "/")
}

// makePipeline creates a PipelineRun and substitutes LighthouseJob managed pipeline resources with ResourceSpec instead of ResourceRef
// so that we don't have to take care of potentially dangling created pipeline resources.
func makePipelineRun(ctx context.Context, lj v1alpha1.LighthouseJob, breakpoints []*v1alpha1.LighthouseBreakpoint, namespace string, logger *logrus.Entry, idGen buildIDGenerator, c client.Reader) (*tektonv1beta1.PipelineRun, error) {
	// First validate.
	if lj.Spec.PipelineRunSpec == nil {
		return nil, errors.New("no PipelineSpec defined")
	}

	buildID := idGen.GenerateBuildID()
	if buildID == "" {
		return nil, errors.New("empty BuildID in status")
	}

	prLabels, annotations := jobutil.LabelsAndAnnotationsForJob(lj, buildID)
	specCopy := lj.Spec.PipelineRunSpec.DeepCopy()
	generateName := jobutil.GenerateName(&lj.Spec)
	p := tektonv1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Annotations:  annotations,
			GenerateName: generateName,
			Namespace:    namespace,
			Labels:       prLabels,
		},
		Spec: *specCopy,
	}
	// Set a default timeout of 1 day if no timeout is specified
	if p.Spec.Timeout == nil {
		p.Spec.Timeout = &metav1.Duration{Duration: 24 * time.Hour}
	}

	// Add parameters instead of env vars.
	env := lj.Spec.GetEnvVars()
	env[v1alpha1.BuildIDEnv] = buildID
	env[v1alpha1.RepoURLEnv] = lj.Spec.Refs.CloneURI
	var batchedRefsVals []string
	for _, pull := range lj.Spec.Refs.Pulls {
		if pull.Ref != "" {
			batchedRefsVals = append(batchedRefsVals, pull.Ref)
		}
	}
	if len(batchedRefsVals) > 0 {
		env[v1alpha1.PullPullRefEnv] = strings.Join(batchedRefsVals, " ")
	}
	if len(lj.Spec.PipelineRunParams) > 0 {
		payload := map[string]interface{}{
			"Refs": lj.Spec.Refs,
		}
		for _, param := range lj.Spec.PipelineRunParams {
			parsedTemplate, err := template.New(param.Name).Parse(param.ValueTemplate)
			if err != nil {
				return nil, err
			}
			var msgBuffer bytes.Buffer
			err = parsedTemplate.Execute(&msgBuffer, payload)
			if err != nil {
				return nil, err
			}
			env[param.Name] = msgBuffer.String()
		}
	} else {
		paramNames, err := determineGitCloneOrMergeTaskParams(ctx, &p, c)
		if err != nil {
			return nil, err
		}
		if paramNames == nil {
			logger.Warnf("git-clone and/or git-batch-merge task parameters not found in Pipeline for PipelineRun, so skipping setting PipelineRun parameters for revision")
		} else {
			env[paramNames.urlParam] = lj.Spec.Refs.CloneURI
			if paramNames.revParam != "" {
				if len(lj.Spec.Refs.Pulls) > 0 {
					env[paramNames.revParam] = lj.Spec.Refs.Pulls[0].SHA
				} else {
					env[paramNames.revParam] = lj.Spec.Refs.BaseRef
				}
			}
			if paramNames.baseRevisionParam != "" {
				env[paramNames.baseRevisionParam] = lj.Spec.Refs.BaseRef
			}
			if paramNames.batchedRefsParam != "" {
				env[paramNames.batchedRefsParam] = strings.Join(batchedRefsVals, " ")
			}
		}
	}
	for _, key := range sets.StringKeySet(env).List() {
		val := env[key]
		// TODO: make this handle existing values/substitutions.
		p.Spec.Params = append(p.Spec.Params, tektonv1beta1.Param{
			Name: key,
			Value: tektonv1beta1.ArrayOrString{
				Type:      tektonv1beta1.ParamTypeString,
				StringVal: val,
			},
		})
	}

	// lets apply any breakpoints...
	taskNames := findAllTaskNames(&p)
	for _, taskName := range taskNames {
		filterValues := resolveBreakpointFilter(&p, taskName, lj)
		debug := filterValues.ResolveDebug(breakpoints)
		if debug != nil {
			found := false
			for i := range p.Spec.TaskRunSpecs {
				trs := &p.Spec.TaskRunSpecs[i]
				if trs.PipelineTaskName == taskName {
					trs.Debug = debug
					found = true
					break
				}
			}
			if !found {
				p.Spec.TaskRunSpecs = append(p.Spec.TaskRunSpecs, tektonv1beta1.PipelineTaskRunSpec{
					PipelineTaskName: taskName,
					Debug:            debug,
				})
			}
		}
	}
	return &p, nil
}

// findAllTaskNames finds all the task names from either the PR.Spec.TaskRunspecs or PR.Spec.PipelineSpec.Tasks
// so we know which task names may require debug
func findAllTaskNames(p *tektonv1beta1.PipelineRun) []string {
	var answer []string

	for i := range p.Spec.TaskRunSpecs {
		trs := &p.Spec.TaskRunSpecs[i]
		answer = append(answer, trs.PipelineTaskName)
	}
	ps := p.Spec.PipelineSpec
	if ps != nil {
		for i := range ps.Tasks {
			t := &ps.Tasks[i]
			if t.Name != "" && util.StringArrayIndex(answer, t.Name) < 0 {
				answer = append(answer, t.Name)
			}
		}
	}
	return answer
}

func resolveBreakpointFilter(p *tektonv1beta1.PipelineRun, name string, lj v1alpha1.LighthouseJob) *v1alpha1.LighthousePipelineFilter {
	filterValues := &v1alpha1.LighthousePipelineFilter{
		Type:    lj.Spec.Type,
		Context: lj.Spec.Context,
	}
	branch := ""
	if lj.Labels != nil {
		branch = lj.Labels[util.BranchLabel]
	}
	refs := lj.Spec.Refs
	if refs != nil {
		filterValues.Owner = refs.Org
		filterValues.Repository = refs.Repo
		if branch == "" {
			for _, p := range refs.Pulls {
				if p.Number > 0 {
					branch = fmt.Sprintf("PR-%d", p.Number)
					break
				}
			}
		}
		if branch == "" {
			branch = refs.BaseRef
		}
	}
	filterValues.Branch = branch

	if lj.Labels != nil {
		if filterValues.Owner == "" {
			filterValues.Owner = lj.Labels[util.OrgLabel]
		}
		if filterValues.Repository == "" {
			filterValues.Repository = lj.Labels[util.RepoLabel]
		}
	}
	return filterValues
}

type gitTaskParamNames struct {
	urlParam          string
	revParam          string
	batchedRefsParam  string
	baseRevisionParam string
}

func determineGitCloneOrMergeTaskParams(ctx context.Context, pr *tektonv1beta1.PipelineRun, c client.Reader) (*gitTaskParamNames, error) {
	if pr == nil {
		return nil, errors.New("provided PipelineRun is nil")
	}

	if pr.Spec.PipelineSpec == nil && pr.Spec.PipelineRef == nil {
		return nil, errors.New("neither PipelineSpec nor PipelineRef specified for PipelineRun")
	}
	var pipelineSpec *tektonv1beta1.PipelineSpec

	if pr.Spec.PipelineSpec != nil {
		pipelineSpec = pr.Spec.PipelineSpec
	} else {
		pipeline := tektonv1beta1.Pipeline{ObjectMeta: metav1.ObjectMeta{Name: pr.Spec.PipelineRef.Name, Namespace: pr.Namespace}}
		key := client.ObjectKeyFromObject(&pipeline)
		err := c.Get(ctx, key, &pipeline)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to find Pipeline %s for PipelineRun", pr.Spec.PipelineRef.Name)
		}
		pipelineSpec = &pipeline.Spec
	}

	paramNames := &gitTaskParamNames{}

	for _, task := range pipelineSpec.Tasks {
		if task.TaskRef != nil {
			if task.TaskRef.Name == gitCloneCatalogTaskName {
				for _, p := range task.Params {
					if p.Name == gitCloneURLParam && p.Value.Type == tektonv1beta1.ParamTypeString {
						paramNames.urlParam = extractPipelineParamFromTaskParamValue(p.Value.StringVal)
					}
					if p.Name == gitCloneRevisionParam && p.Value.Type == tektonv1beta1.ParamTypeString {
						paramNames.revParam = extractPipelineParamFromTaskParamValue(p.Value.StringVal)
					}
				}

				if paramNames.urlParam != "" && paramNames.revParam != "" {
					return paramNames, nil
				}
			}
			if task.TaskRef.Name == gitMergeCatalogTaskName {
				for _, p := range task.Params {
					if p.Name == gitCloneURLParam && p.Value.Type == tektonv1beta1.ParamTypeString {
						paramNames.urlParam = extractPipelineParamFromTaskParamValue(p.Value.StringVal)
					}
					if p.Name == gitCloneRevisionParam && p.Value.Type == tektonv1beta1.ParamTypeString {
						paramNames.baseRevisionParam = extractPipelineParamFromTaskParamValue(p.Value.StringVal)
					}
					if p.Name == gitMergeBatchRefsParam && p.Value.Type == tektonv1beta1.ParamTypeString {
						paramNames.batchedRefsParam = extractPipelineParamFromTaskParamValue(p.Value.StringVal)
					}
				}

				if paramNames.urlParam != "" && paramNames.batchedRefsParam != "" {
					return paramNames, nil
				}

			}
		}
	}

	return nil, nil
}

func extractPipelineParamFromTaskParamValue(taskParam string) string {
	if strings.HasPrefix(taskParam, "$(params.") && strings.HasSuffix(taskParam, ")") {
		return strings.TrimPrefix(strings.TrimSuffix(taskParam, ")"), "$(params.")
	}
	return ""
}
