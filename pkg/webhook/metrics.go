/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package webhook

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Define all metrics for webhooks here.
	webhookCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lighthouse_webhook_counter",
		Help: "A counter of the webhooks made to lighthouse.",
	}, []string{"event_type"})
	responseCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lighthouse_webhook_response_codes",
		Help: "A counter of the different responses hook has responded to webhooks with.",
	}, []string{"response_code"})
)

// Metrics is a set of metrics gathered by hook.
type Metrics struct {
	WebhookCounter  *prometheus.CounterVec
	ResponseCounter *prometheus.CounterVec
}

// NewMetrics creates a new set of metrics for the hook server.
func NewMetrics() *Metrics {
	return &Metrics{
		WebhookCounter:  webhookCounter,
		ResponseCounter: responseCounter,
	}
}
