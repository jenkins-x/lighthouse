module github.com/jenkins-x/lighthouse

require (
	code.gitea.io/sdk/gitea v0.13.1-0.20201217044651-7ddbf1a0151e // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible
	github.com/NYTimes/gziphandler v0.0.0-20170623195520-56545f4a5d46
	github.com/bwmarrin/snowflake v0.0.0
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/evanphx/json-patch v4.5.0+incompatible
	github.com/go-stack/stack v1.8.0
	github.com/google/go-cmp v0.4.1
	github.com/gorilla/sessions v1.2.0
	github.com/hashicorp/go-multierror v1.1.0
	github.com/hashicorp/golang-lru v0.5.4
	github.com/jenkins-x/go-scm v1.5.209
	github.com/mattn/go-zglob v0.0.1
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.5.0
	github.com/satori/go.uuid v1.2.1-0.20180103174451-36e9d2ebbde5
	github.com/shurcooL/githubv4 v0.0.0-20191102174205-af46314aec7b
	github.com/sirupsen/logrus v1.6.0
	github.com/stretchr/testify v1.6.1
	github.com/tektoncd/pipeline v0.14.2
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/time v0.0.0-20200416051211-89c76fbcd5d1 // indirect
	google.golang.org/grpc v1.28.1 // indirect
	gopkg.in/robfig/cron.v2 v2.0.0-20150107220207-be2e0b0deed5
	k8s.io/api v0.17.6
	k8s.io/apimachinery v0.17.6
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/utils v0.0.0-20200124190032-861946025e34
	knative.dev/pkg v0.0.0-20200702222342-ea4d6e985ba0
	sigs.k8s.io/controller-runtime v0.5.0
	sigs.k8s.io/yaml v1.2.0
)

replace k8s.io/client-go => k8s.io/client-go v0.17.6

// gomodules.xyz breaks in Athens proxying
replace gomodules.xyz/jsonpatch/v2 => github.com/gomodules/jsonpatch/v2 v2.1.0

// vbom.ml doesn't actually exist any more
replace vbom.ml/util => github.com/fvbommel/util v0.0.0-20180919145318-efcd4e0f9787

go 1.13
