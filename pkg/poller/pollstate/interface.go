package pollstate

type Interface interface {
	IsNew(repository, operation, values string) (bool, error)
}
