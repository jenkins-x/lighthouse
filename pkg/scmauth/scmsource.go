package scmauth

import (
	"context"

	"github.com/jenkins-x/go-scm/scm"
)

// ForOwner adapts src to a go-scm token source bound to owner, since go-scm resolves
// credentials per request without knowing which owner the request is for.
func ForOwner(src TokenSource, owner string) scm.TokenSource {
	return &ownerBoundSource{src: src, owner: owner}
}

type ownerBoundSource struct {
	src   TokenSource
	owner string
}

func (s *ownerBoundSource) Token(context.Context) (*scm.Token, error) {
	token, err := s.src.Token(s.owner)
	if err != nil {
		return nil, err
	}
	return &scm.Token{Token: token}, nil
}
