package pollstate

type Interface interface {
	IsNew(repository, operation, values string) (bool, error)
	Invalidate(repository, operation, invalidValue string)
}
