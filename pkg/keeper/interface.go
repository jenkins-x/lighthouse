package keeper

import (
	"net/http"

	"github.com/jenkins-x/lighthouse/pkg/keeper/history"
)

// Controller the interface for all keeper controllers
type Controller interface {
	Sync() error
	Shutdown()
	GetPools() []Pool
	ServeHTTP(w http.ResponseWriter, r *http.Request)
	GetHistory() *history.History
}
