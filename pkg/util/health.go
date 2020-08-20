package util

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jenkins-x/lighthouse/pkg/interrupts"
)

const healthPort = 8081

// Health keeps a request multiplexer for health liveness and readiness endpoints
type Health struct {
	healthMux *http.ServeMux
}

// NewHealth creates a new health request multiplexer and starts serving the liveness endpoint
// on the given port
func NewHealth() *Health {
	healthMux := http.NewServeMux()
	healthMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "OK") })
	server := &http.Server{Addr: ":" + strconv.Itoa(healthPort), Handler: healthMux}
	interrupts.ListenAndServe(server, 5*time.Second)
	return &Health{
		healthMux: healthMux,
	}
}

// ServeReady starts serving the readiness endpoint
func (h *Health) ServeReady() {
	h.healthMux.HandleFunc("/healthz/ready", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "OK") })
}
