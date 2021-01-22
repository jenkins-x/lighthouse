package inrepo

import (
	"fmt"
	"github.com/jenkins-x/go-scm/scm"
	"io/ioutil"
	"net/url"
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/jenkins-x/lighthouse/pkg/filebrowser"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/pkg/errors"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	v1 "k8s.io/api/core/v1"
)

// UsesResolver resolves the `uses:` URI syntax
type UsesResolver struct {
	Client           filebrowser.Interface
	OwnerName        string
	RepoName         string
	SHA              string
	Dir              string
	Message          string
	DefaultValues    *DefaultValues
	LocalFileResolve bool

	cache map[string]*tektonv1beta1.PipelineRun
}

var (
	// VersionStreamVersions allows you to register version stream values of ful repository names in the format
	// `owner/name` mapping to the version SHA/branch/tag
	VersionStreamVersions = map[string]string{}
)

// UsesSteps lets resolve the sourceURI to a PipelineRun and find the step or steps
// for the given task name and/or step name then lets apply any overrides from the step
func (r *UsesResolver) UsesSteps(sourceURI string, taskName string, step tektonv1beta1.Step) ([]tektonv1beta1.Step, error) {
	if r.cache == nil {
		r.cache = map[string]*tektonv1beta1.PipelineRun{}
	}
	pr := r.cache[sourceURI]
	if pr == nil {
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
		r.cache[sourceURI] = pr
	}

	return r.findSteps(sourceURI, pr, taskName, step)
}

// GetData gets the data from the given source URI
func (r *UsesResolver) GetData(path string, ignoreNotExist bool) ([]byte, error) {
	var data []byte
	_, err := url.ParseRequestURI(path)
	if err == nil {
		data, err = getPipelineFromURL(path)
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
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read file %s", path)
		}
		return data, nil
	}
	if gitURI != nil {
		owner = gitURI.Owner
		repo = gitURI.Repository
		path = gitURI.Path
		sha = resolveCustomSha(owner, repo, gitURI.SHA)
	}
	data, err = r.Client.GetFile(owner, repo, path, sha)
	if err != nil && IsScmNotFound(err) {
		err = nil
	}
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find file %s in repo %s/%s with sha %s", path, owner, repo, sha)
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

func (r *UsesResolver) findSteps(sourceURI string, pr *tektonv1beta1.PipelineRun, taskName string, step tektonv1beta1.Step) ([]tektonv1beta1.Step, error) {
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

func (r *UsesResolver) findTaskStep(sourceURI string, task tektonv1beta1.PipelineTask, step tektonv1beta1.Step) ([]tektonv1beta1.Step, error) {
	ts := task.TaskSpec
	if ts == nil {
		return nil, errors.Errorf("source URI %s has no task spec for task %s", sourceURI, task.Name)
	}
	name := step.Name
	if name == "" {
		return ts.Steps, nil
	}

	idx := strings.Index(name, ":")
	suffix := ""
	if idx > 0 {
		suffix = name[idx+1:]
		name = name[0:idx]
	}

	for i := range ts.Steps {
		s := &ts.Steps[i]
		if s.Name == name {
			replaceStep := *s
			OverrideStep(&replaceStep, &step)
			if suffix != "" {
				replaceStep.Name = name + "-" + suffix
			}
			return []tektonv1beta1.Step{replaceStep}, nil
		}
	}
	return nil, errors.Errorf("source URI %s task %s has no step named %s", sourceURI, task.Name, name)
}

// OverrideStep overrides the step with the given overrides
func OverrideStep(step *tektonv1beta1.Step, override *tektonv1beta1.Step) {
	if len(override.Command) > 0 {
		step.Script = override.Script
		step.Command = override.Command
		step.Args = override.Args
	}
	if override.Script != "" {
		step.Script = override.Script
		step.Command = nil
		step.Args = nil
	}
	if override.Timeout != nil {
		step.Timeout = override.Timeout
	}
	if override.WorkingDir != "" {
		step.WorkingDir = override.WorkingDir
	}
	if string(override.ImagePullPolicy) != "" {
		step.ImagePullPolicy = override.ImagePullPolicy
	}
	step.Env = OverrideEnv(step.Env, override.Env)
	step.EnvFrom = OverrideEnvFrom(step.EnvFrom, override.EnvFrom)
	step.VolumeMounts = OverrideVolumeMounts(step.VolumeMounts, override.VolumeMounts)
}

// OverrideEnv override either replaces or adds the given env vars
func OverrideEnv(from []v1.EnvVar, overrides []v1.EnvVar) []v1.EnvVar {
	for _, override := range overrides {
		found := false
		for i := range from {
			f := &from[i]
			if f.Name == override.Name {
				found = true
				*f = override
				break
			}
		}
		if !found {
			from = append(from, override)
		}
	}
	return from
}

// OverrideEnvFrom override either replaces or adds the given env froms
func OverrideEnvFrom(from []v1.EnvFromSource, overrides []v1.EnvFromSource) []v1.EnvFromSource {
	for _, override := range overrides {
		found := false
		for i := range from {
			f := &from[i]
			if f.ConfigMapRef != nil && override.ConfigMapRef != nil && f.ConfigMapRef.Name == override.ConfigMapRef.Name {
				found = true
				*f = override
				break
			}
			if f.SecretRef != nil && override.SecretRef != nil && f.SecretRef.Name == override.SecretRef.Name {
				found = true
				*f = override
				break
			}
		}
		if !found {
			from = append(from, override)
		}
	}
	return from
}

// OverrideVolumeMounts override either replaces or adds the given volume mounts
func OverrideVolumeMounts(from []v1.VolumeMount, overrides []v1.VolumeMount) []v1.VolumeMount {
	for _, override := range overrides {
		found := false
		for i := range from {
			f := &from[i]
			if f.Name == override.Name {
				found = true
				*f = override
				break
			}
		}
		if !found {
			from = append(from, override)
		}
	}
	return from
}
