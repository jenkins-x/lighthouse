package builder

// Builder the interface for the pipeline builder
type Builder interface {
	Create(*ProwJob) (*ProwJob, error)
}
