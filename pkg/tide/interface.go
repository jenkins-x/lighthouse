package tide

import (
	"net/http"

	"github.com/jenkins-x/lighthouse/pkg/tide/history"
)

type TideController interface {
	Sync() error
	Shutdown()
	GetPools() []Pool
	ServeHTTP(w http.ResponseWriter, r *http.Request)
	GetHistory() *history.History
}
