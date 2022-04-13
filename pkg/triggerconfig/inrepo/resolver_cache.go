package inrepo

import (
	"sync"

	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

const DELIMIER = "#"

// ResolverCache a cache of data and pipelines to minimise
// the git cloning with in repo configurations
type ResolverCache struct {
	lock          sync.RWMutex
	pipelineCache map[string]*tektonv1beta1.PipelineRun
	dataCache     map[string][]byte
}

// NewResolverCache creates a new resolver cache
func NewResolverCache() *ResolverCache {
	return &ResolverCache{
		pipelineCache: map[string]*tektonv1beta1.PipelineRun{},
		dataCache:     map[string][]byte{},
	}
}

// GetData gets data from the cache if available or returns nil
func (c *ResolverCache) GetData(sourceURI, ref string) []byte {
	if c == nil || sourceURI == "" {
		return nil
	}
	var answer []byte
	c.lock.Lock()
	answer = c.dataCache[sourceURI+DELIMIER+ref]
	c.lock.Unlock()
	return answer
}

// SetData updates the cache
func (c *ResolverCache) SetData(sourceURI, ref string, value []byte) {
	if c == nil || len(value) == 0 {
		return
	}
	c.lock.Lock()
	c.dataCache[sourceURI+DELIMIER+ref] = value
	c.lock.Unlock()
}

// GetPipelineRun gets the PipelineRun from the cache if available or returns nil
func (c *ResolverCache) GetPipelineRun(sourceURI, ref string) *tektonv1beta1.PipelineRun {
	if c == nil || sourceURI == "" {
		return nil
	}
	var answer *tektonv1beta1.PipelineRun
	c.lock.Lock()
	answer = c.pipelineCache[sourceURI+DELIMIER+ref]
	c.lock.Unlock()
	return answer
}

// SetPipelineRun updates the cache
func (c *ResolverCache) SetPipelineRun(sourceURI string, ref string, value *tektonv1beta1.PipelineRun) {
	if c == nil || value == nil {
		return
	}
	c.lock.Lock()
	c.pipelineCache[sourceURI+DELIMIER+ref] = value
	c.lock.Unlock()
}
