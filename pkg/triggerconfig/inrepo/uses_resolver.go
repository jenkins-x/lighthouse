package inrepo

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jenkins-x/go-scm/scm"

	"github.com/sirupsen/logrus"

	"github.com/jenkins-x/lighthouse/pkg/filebrowser"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/pkg/errors"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

// UsesResolver resolves the `uses:` URI syntax
type UsesResolver struct {
	FileBrowsers     *filebrowser.FileBrowsers
	FetchCache       filebrowser.FetchCache
	Cache            *ResolverCache
	OwnerName        string
	RepoName         string
	SHA              string
	Dir              string
	Message          string
	DefaultValues    *DefaultValues
	LocalFileResolve bool
}

var (
	// VersionStreamVersions allows you to register version stream values of ful repository names in the format
	// `owner/name` mapping to the version SHA/branch/tag
	VersionStreamVersions = map[string]string{}

	ignoreUsesCache = os.Getenv("NO_USES_CACHE") == "true"
)

// UsesSteps lets resolve the sourceURI to a PipelineRun and find the step or steps
// for the given task name and/or step name then lets apply any overrides from the step
func (r *UsesResolver) UsesSteps(sourceURI string, taskName string, step pipelinev1beta1.Step, ts *pipelinev1beta1.TaskSpec, loc *UseLocation) ([]pipelinev1beta1.Step, error) {
	pr := r.Cache.GetPipelineRun(sourceURI, r.SHA)
	if pr == nil || ignoreUsesCache {
		data, err := r.GetData(sourceURI, false)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to load URI %s", sourceURI)
		}
		if len(data) == 0 {
			return nil, errors.Errorf("source URI not found: %s", sourceURI)
		}

		pr, err = LoadTektonResourceAsPipelineRun(r, data)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to resolve %s", sourceURI)
		}
		if pr == nil {
			return nil, errors.Errorf("no PipelineRun for URI %s", sourceURI)
		}
		r.Cache.SetPipelineRun(sourceURI, r.SHA, pr)
	}

	useTS, err := r.findSteps(sourceURI, pr.DeepCopy(), taskName, step)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to ")
	}

	originalSteps := ts.Steps
	steps := useTS.Steps
	OverrideTaskSpec(useTS, ts)

	// lets preserve any parameters, results, workspaces on the ts before overriding...
	ctx := context.TODO()
	useTS.Params = useParameterSpecs(ctx, useTS.Params, ts.Params, nil)
	useTS.Results = useResults(useTS.Results, ts.Results)
	useTS.Workspaces = useWorkspaces(useTS.Workspaces, ts.Workspaces)
	useTS.Sidecars = useSidecars(useTS.Sidecars, ts.Sidecars)
	*ts = *useTS
	ts.Steps = originalSteps

	UseParametersAndResults(ctx, loc, useTS)
	return steps, nil
}

// GetData gets the data from the given source URI
func (r *UsesResolver) GetData(path string, ignoreNotExist bool) ([]byte, error) {
	data := r.Cache.GetData(path, r.SHA)
	if len(data) > 0 {
		return data, nil
	}

	if strings.Contains(path, "://") {
		data, err := getPipelineFromURL(path)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get pipeline from URL %s ", path)
		}
		return data, nil
	}

	owner := r.OwnerName
	repo := r.RepoName
	sha := r.SHA

	gitURI, err := ParseGitURI(path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse git URI %s", path)
	}
	if gitURI == nil && r.LocalFileResolve {
		if ignoreNotExist {
			exists, err := util.FileExists(path)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to check file exists %s", path)
			}
			if !exists {
				return nil, nil
			}
		}
		/* #nosec */
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read file %s", path)
		}
		return data, nil
	}
	fb := r.FileBrowsers.LighthouseGitFileBrowser()
	if gitURI != nil {
		data := r.lookupDataCache(path)
		if len(data) > 0 {
			return data, nil
		}

		owner = gitURI.Owner
		repo = gitURI.Repository
		path = gitURI.Path
		sha = resolveCustomSha(owner, repo, gitURI.SHA)

		fb = r.FileBrowsers.GetFileBrowser(gitURI.Server)
		if fb == nil {
			return nil, errors.Errorf("could not find git file browser for server %s in uses: git URI %s", gitURI.Server, gitURI.String())
		}
	}
	data, err = fb.GetFile(owner, repo, path, sha, r.FetchCache)
	if err != nil && IsScmNotFound(err) {
		err = nil
	}
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find file %s in repo %s/%s with sha %s", path, owner, repo, sha)
	}
	if gitURI != nil {
		r.Cache.SetData(path, r.SHA, data)
	}
	return data, nil
}

// lets allow version stream versions to be exposed by environment variables
func resolveCustomSha(owner string, repo string, sha string) string {
	if sha != "versionStream" {
		return sha
	}
	fullName := scm.Join(owner, repo)
	value := VersionStreamVersions[fullName]
	envVar := VersionStreamEnvVar(owner, repo)
	if value == "" {
		value = os.Getenv(envVar)
	}
	if value != "" {
		return value
	}
	logrus.WithFields(map[string]interface{}{
		"Owner": owner,
		"Repo":  repo,
	}).Warnf("no version stream version environment variable: %s", envVar)
	return "HEAD"
}

// VersionStreamEnvVar creates an environment variable name
func VersionStreamEnvVar(owner string, repo string) string {
	envVar := strings.ToUpper(fmt.Sprintf("LIGHTHOUSE_VERSIONSTREAM_%s_%s", owner, repo))
	envVar = strings.ReplaceAll(envVar, "-", "_")
	envVar = strings.ReplaceAll(envVar, " ", "_")
	envVar = strings.ReplaceAll(envVar, ".", "_")
	return envVar
}

func (r *UsesResolver) findSteps(sourceURI string, pr *pipelinev1beta1.PipelineRun, taskName string, step pipelinev1beta1.Step) (*pipelinev1beta1.TaskSpec, error) {
	if pr.Spec.PipelineSpec == nil {
		return nil, errors.Errorf("source URI %s has no spec.pipelineSpec", sourceURI)
	}
	pipelineTasks := pr.Spec.PipelineSpec.Tasks
	switch len(pipelineTasks) {
	case 0:
		return nil, errors.Errorf("source URI %s has no spec.pipelineSpec.tasks", sourceURI)
	case 1:
		return r.findTaskStep(sourceURI, pipelineTasks[0], step)

	default:
		for _, task := range pipelineTasks {
			if task.Name == taskName {
				return r.findTaskStep(sourceURI, task, step)
			}
		}
		return nil, errors.Errorf("source URI %s has multiple spec.pipelineSpec.tasks but none match task name %s", sourceURI, taskName)
	}
}

func (r *UsesResolver) findTaskStep(sourceURI string, task pipelinev1beta1.PipelineTask, step pipelinev1beta1.Step) (*pipelinev1beta1.TaskSpec, error) {
	ts := task.TaskSpec
	if ts == nil {
		return nil, errors.Errorf("source URI %s has no task spec for task %s", sourceURI, task.Name)
	}
	name := step.Name
	if name == "" {
		return &ts.TaskSpec, nil
	}

	idx := strings.Index(name, ":")
	suffix := ""
	if idx > 0 {
		suffix = name[idx+1:]
		name = name[0:idx]
	}

	taskSpec := task.TaskSpec.TaskSpec
	for i := range ts.Steps {
		s := &ts.Steps[i]
		if s.Name == name {
			replaceStep := *s
			OverrideStep(&replaceStep, &step)
			if suffix != "" {
				replaceStep.Name = name + "-" + suffix
			}
			taskSpec.Steps = []pipelinev1beta1.Step{replaceStep}
			return &taskSpec, nil
		}
	}
	return nil, errors.Errorf("source URI %s task %s has no step named %s", sourceURI, task.Name, name)
}

func (r *UsesResolver) lookupDataCache(path string) []byte {
	return nil
}
