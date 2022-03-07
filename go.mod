module github.com/jenkins-x/lighthouse

require (
	github.com/Azure/go-autorest v14.2.0+incompatible
	github.com/NYTimes/gziphandler v0.0.0-20170623195520-56545f4a5d46
	github.com/bwmarrin/snowflake v0.0.0
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/evanphx/json-patch v4.9.0+incompatible
	github.com/go-stack/stack v1.8.0
	github.com/google/go-cmp v0.5.6
	github.com/gorilla/sessions v1.2.0
	github.com/h2non/gock v1.0.9
	github.com/hashicorp/go-multierror v1.1.0
	github.com/hashicorp/golang-lru v0.5.4
	github.com/jenkins-x/go-scm v1.11.3
	github.com/mattn/go-zglob v0.0.1
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.3
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/shurcooL/githubv4 v0.0.0-20191102174205-af46314aec7b
	github.com/sirupsen/logrus v1.7.0
	github.com/stretchr/testify v1.7.0
	github.com/tektoncd/pipeline v0.26.0
	golang.org/x/oauth2 v0.0.0-20210628180205-a41e5a781914
	gopkg.in/robfig/cron.v2 v2.0.0-20150107220207-be2e0b0deed5
	k8s.io/api v0.20.7
	k8s.io/apimachinery v0.20.7
	k8s.io/client-go v0.20.7
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009
	knative.dev/pkg v0.0.0-20210730172132-bb4aaf09c430
	sigs.k8s.io/controller-runtime v0.8.0
	sigs.k8s.io/yaml v1.2.0
)

replace (
	// Knative deps (release-0.20)
	contrib.go.opencensus.io/exporter/stackdriver => contrib.go.opencensus.io/exporter/stackdriver v0.13.4
	github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v38.2.0+incompatible

	// lets override the go-scm version from tektoncd
	github.com/jenkins-x/go-scm => github.com/jenkins-x/go-scm v1.11.3

	// for the PipelineRun debug fix see: https://github.com/tektoncd/pipeline/pull/4145
	github.com/tektoncd/pipeline => github.com/jstrachan/pipeline v0.21.1-0.20210811150720-45a86a5488af

	// gomodules.xyz breaks in Athens proxying
	gomodules.xyz/jsonpatch/v2 => github.com/gomodules/jsonpatch/v2 v2.2.0
	k8s.io/api => k8s.io/api v0.20.7
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.7
	k8s.io/client-go => k8s.io/client-go v0.20.7
	knative.dev/pkg => knative.dev/pkg v0.0.0-20210730172132-bb4aaf09c430

	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.8.0
)

go 1.15
