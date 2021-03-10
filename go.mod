module github.com/jenkins-x/lighthouse

require (
	contrib.go.opencensus.io/exporter/stackdriver v0.13.5 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.18 // indirect
	github.com/NYTimes/gziphandler v0.0.0-20170623195520-56545f4a5d46
	github.com/aws/aws-sdk-go v1.31.15 // indirect
	github.com/bwmarrin/snowflake v0.0.0
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/emicklei/go-restful v2.9.6+incompatible // indirect
	github.com/evanphx/json-patch v4.9.0+incompatible
	github.com/go-openapi/spec v0.19.15 // indirect
	github.com/go-openapi/swag v0.19.14 // indirect
	github.com/go-stack/stack v1.8.0
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/go-cmp v0.5.5
	github.com/google/go-containerregistry v0.4.1 // indirect
	github.com/google/uuid v1.1.5 // indirect
	github.com/googleapis/gnostic v0.5.4 // indirect
	github.com/gorilla/sessions v1.2.1
	github.com/h2non/gock v1.0.9
	github.com/hashicorp/go-multierror v1.1.0
	github.com/hashicorp/golang-lru v0.5.4
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/jenkins-x/go-scm v1.5.225
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-zglob v0.0.3
	github.com/mitchellh/reflectwalk v1.0.1 // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.5
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.8.0
	github.com/satori/go.uuid v1.2.1-0.20180103174451-36e9d2ebbde5
	github.com/shurcooL/githubv4 v0.0.0-20191102174205-af46314aec7b
	github.com/sirupsen/logrus v1.6.0
	github.com/stretchr/testify v1.6.1
	github.com/tektoncd/pipeline v0.20.1
	go.opencensus.io v0.22.6 // indirect
	golang.org/x/oauth2 v0.0.0-20201208152858-08078c50e5b5
	golang.org/x/text v0.3.5 // indirect
	google.golang.org/grpc v1.34.1 // indirect
	gopkg.in/robfig/cron.v2 v2.0.0-20150107220207-be2e0b0deed5
	k8s.io/api v0.20.4
	k8s.io/apiextensions-apiserver v0.20.4 // indirect
	k8s.io/apimachinery v0.20.4
	k8s.io/client-go v0.20.4
	k8s.io/utils v0.0.0-20201110183641-67b214c5f920
	knative.dev/pkg v0.0.0-20210107022335-51c72e24c179
	sigs.k8s.io/controller-runtime v0.8.3
	sigs.k8s.io/structured-merge-diff/v4 v4.0.3 // indirect
	sigs.k8s.io/yaml v1.2.0
)

replace (
	// lets override the go-scm version from tektoncd
	github.com/jenkins-x/go-scm => github.com/jenkins-x/go-scm v1.5.223
	github.com/tektoncd/pipeline => github.com/jenkins-x/pipeline v0.3.2-0.20210223153617-0d1186b27496

	// gomodules.xyz breaks in Athens proxying
	gomodules.xyz/jsonpatch/v2 => github.com/gomodules/jsonpatch/v2 v2.1.0
	k8s.io/api => k8s.io/api v0.20.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.2
	k8s.io/client-go => k8s.io/client-go v0.20.2
	knative.dev/pkg => github.com/jstrachan/pkg v0.0.0-20210118084935-c7bdd6c14bd0

	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.8.0
)

go 1.15
