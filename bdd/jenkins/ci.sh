#!/usr/bin/env bash
#
# Script for running Lighthouse + Jenkins BDD tests.
#
# Arguments:
# none

# set -x
set -e
shopt -s expand_aliases

#######################################
# Writes usage message
#######################################
function usage {
    echo "usage: $programname [-l] "
    echo "  -l     Skips all cluster creation steps and executes tests against currently connected cluster."
    exit 1
}

###############################################################################
# Helper to keep track of called functions
###############################################################################
function exe() {
	start=$(date +%s)
	echo "###############################################################################"
	echo "\$ $*"
	echo "###############################################################################"
	"$@"
	end=$(date +%s)
	runtime=$((end - start))
	echo "exec time: $(printf '%dh:%dm:%ds\n' $(($runtime / 3600)) $(($runtime % 3600 / 60)) $(($runtime % 60)))"
	echo -e "\n\n"
}

###############################################################################
# Helper to execute a command and only print stdout and stderr if the command
# does not complete successfully
#
# Arguments:
# The command to execute
###############################################################################
function suppress {
  temp=$(mktemp)
  ${1+"$@"} > "${temp}" 2>&1 || cat "${temp}"
  rm "${temp}"
}

###############################################################################
# Curls the specified URL until a HTTP 200 is returned or a timeout occurs.
#
# Arguments:
#   $1 - The URL to hit
#   $2 - Timeout in seconds
# Returns:
#   Returns 0 if the service returns a HTTP 200 within the specified timeout,
#   1 otherwise.
###############################################################################
function wait_for_http_status_200 {
  url=$1
  timeout=$2

	i=0
	upper=$((timeout/10))
	while [ $i -lt $upper ]; do
		if [ "$(curl -s -o /dev/null -w "%{http_code}" "$url")" == "200" ]
		then
			echo "$url returns 200"
			return 0
		fi
		i=$((i + 1))
		sleep 10
	done

	return 1
}

function run_helm() {
  # in order to run this script locally I might
	if command -v helm3 &> /dev/null; then
   helm3 "$@"
  else
   helm "$@"
  fi
}

###############################################################################
# Ensures that all required environment variables are provided.
# Exists the execution in case one more more are missing.
###############################################################################
function ensure_environment() {
	missingEnvVarMessages=()
	if [ -z "${VERSION}" ]; then
		missingEnvVarMessages+=("VERSION for Lighthouse images not set")
	fi

	if [ -z "${E2E_PRIMARY_SCM_USER}" ]; then
		missingEnvVarMessages+=("E2E_PRIMARY_SCM_USER for bot git provider user not set")
	fi

	if [ -z "${E2E_APPROVER_SCM_USER}" ]; then
		missingEnvVarMessages+=("E2E_APPROVER_SCM_USER for approver git provider user not set")
	fi

	if [ -z "${E2E_PRIMARY_SCM_TOKEN}" ]; then
		missingEnvVarMessages+=("E2E_PRIMARY_SCM_TOKEN for bot git provider token not set")
	fi

	if [ -z "${E2E_APPROVER_SCM_TOKEN}" ]; then
		missingEnvVarMessages+=("E2E_APPROVER_SCM_TOKEN for approver git provider token not set")
	fi

	if [ -z "${E2E_GIT_KIND}" ]; then
		missingEnvVarMessages+=("E2E_GIT_KIND for git flavor (one of 'github', 'gitlab', 'stash') not set")
	fi

	if [ -z "${E2E_GIT_SERVER}" ]; then
		missingEnvVarMessages+=("E2E_GIT_SERVER for git server base URL (i.e., 'https://github.com') not set")
	fi

	if [ -z "${E2E_TEST_NAMESPACE}" ]; then
		missingEnvVarMessages+=("E2E_TEST_NAMESPACE for the Kubernetes test namespace not set")
	fi

	if [ -z "${E2E_JENKINS_HOSTNAME}" ]; then
		missingEnvVarMessages+=("E2E_JENKINS_HOSTNAME for the Jenkins Ingress not set")
	fi

	if [ -z "${E2E_CREATE_LIGHTHOUSE_CRD}" ]; then
		missingEnvVarMessages+=("E2E_CREATE_LIGHTHOUSE_CRD determining CRD import not set")
	fi

	if [ ${#missingEnvVarMessages[@]} -ne 0 ]; then
		echo "ERROR: Missing one or more required environment variables:"
		for msg in "${missingEnvVarMessages[@]}"; do
			echo "${msg}"
		done
		exit 1
	fi

	# Set some default env vars
	if [ -z "${USER}" ]; then
		USER=$(id -u -n)
	fi

	if [ -z "${BRANCH_NAME}" ]; then
		BRANCH_NAME=local
	fi

	if [ -z "${BUILD_ID}" ]; then
		BUILD_ID=1
	fi
}

###############################################################################
# Install any additionally required tools.
# - gomplate for templating the Helm value YAMLs
###############################################################################
function install_tools() {
	suppress go get github.com/hairyhenderson/gomplate/v3/cmd/gomplate
}

###############################################################################
# Creates the GKE test cluster
###############################################################################
function create_cluster() {
	CLUSTER_NAME=$(echo ${BRANCH_NAME,,}-${BUILD_ID,,}-lh-jenkins-e2e)

	echo "Test cluster name: ${CLUSTER_NAME}"

	# Update gcloud so we can do --release-channel
	suppress gcloud components update -q

	# Activate the service account json provided by the pipeline, if it exists
	if [ -n "${GKE_SA}" ]; then
		gcloud auth activate-service-account --key-file "${GKE_SA}"
	else
		echo "GKE_SA environment variable not set, so using current GKE login to create cluster"
	fi

	# Create the cluster with some standard labels and info for cleanup. Minimum version is 1.16.x
	dateLabel=$(date "+%a-%b-%d-%Y-%H-%M-%S")
	gcloud container clusters create "${CLUSTER_NAME}" \
		--zone=europe-west1-c \
		--num-nodes=3 \
		--machine-type n1-standard-2 \
		--enable-autoscaling \
		--min-nodes=3 \
		--max-nodes=5 \
		--scopes=https://www.googleapis.com/auth/cloud-platform,https://www.googleapis.com/auth/compute,https://www.googleapis.com/auth/devstorage.full_control,https://www.googleapis.com/auth/service.management,https://www.googleapis.com/auth/servicecontrol,https://www.googleapis.com/auth/logging.write,https://www.googleapis.com/auth/monitoring \
		--labels="created-by=${USER},create-time=${dateLabel,,},cluster=lh-jenkins-e2e,branch=${BRANCH_NAME,,}" \
		--project=jenkins-x-bdd3 \
		--release-channel=regular

	trap mark_cluster_for_garbage_collection EXIT
}

###############################################################################
# Mark the cluster for garbage collection.
###############################################################################
function mark_cluster_for_garbage_collection() {
  echo ""
  echo "###############################################################################"
  echo "# Marking cluster for garbage collection "
  gcloud container clusters update "${CLUSTER_NAME}" --project=jenkins-x-bdd3 --zone=europe-west1-c --update-labels=delete-me=true
}

###############################################################################
# Configures the test cluster
# Globals:
#   exports EXTERNAL_IP
###############################################################################
function configure_cluster() {
	# Install the nginx ingress controller
	kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v0.34.1/deploy/static/provider/cloud/deploy.yaml

	# And delete the validating webhook config because it causes issues, since helm3 can't access the endpoint from outside the cluster
	kubectl delete validatingwebhookconfigurations.admissionregistration.k8s.io ingress-nginx-admission

	# Capture the external IP since it'll end up getting used in our "domain name", but loop for up to 2 minutes until we get it
	i=0
	while [ $i -lt 12 ]; do
		capturedIp=$(kubectl get svc -n ingress-nginx ingress-nginx-controller -o jsonpath="{.status.loadBalancer.ingress[0].ip}")
		if [ -n "${capturedIp}" ]; then
			export EXTERNAL_IP=${capturedIp}
			break
		fi
		i=$((i + 1))
		sleep 10
	done

	if [ -z "$EXTERNAL_IP" ]
	then
	  echo "Unable to capture cluster IP"
	  exit 1
	fi

	# Create our test namespace and switch to it
	kubectl create namespace "$E2E_TEST_NAMESPACE"
	kubectl config set-context --current --namespace="$E2E_TEST_NAMESPACE"
}

###############################################################################
# Install Jenkins
# Globals:
#   exports E2E_JENKINS_USER
#   exports E2E_JENKINS_PASSWORD
#   exports E2E_JENKINS_URL
###############################################################################
function install_jenkins() {
  export E2E_JENKINS_USER=admin
  E2E_JENKINS_PASSWORD=$(LC_CTYPE=C tr -dc 'A-F0-9' </dev/urandom | head -c10)
  export E2E_JENKINS_PASSWORD

	gomplate -f bdd/jenkins/jenkins-values.yaml.template -o jenkins-values.yaml

	run_helm repo add jenkinsci https://charts.jenkins.io
	run_helm repo update
	run_helm install -f jenkins-values.yaml --version 2.6.4 --namespace "$E2E_TEST_NAMESPACE" jenkins jenkinsci/jenkins

	export E2E_JENKINS_URL=http://"$E2E_JENKINS_HOSTNAME"."$EXTERNAL_IP".nip.io

  if ! wait_for_http_status_200 "$E2E_JENKINS_URL" 360; then
	  echo "###########################"
	  echo "$E2E_JENKINS_URL not ready"
	  echo "###########################"
	  kubectl logs "$(kubectl get pods --selector=app.kubernetes.io/component=jenkins-master -o  jsonpath='{.items[*].metadata.name}')" -c jenkins
	  exit 1
	fi
}

###############################################################################
# Create Jenkins API token
# Globals:
#   exports E2E_JENKINS_API_TOKEN
###############################################################################
function create_jenkins_api_token() {
	crumb=$(curl "$E2E_JENKINS_URL"/crumbIssuer/api/xml?xpath=concat\(//crumbRequestField,%22:%22,//crumb\) \
	  --silent --cookie-jar cookies.txt --user "$E2E_JENKINS_USER:$E2E_JENKINS_PASSWORD")

	echo "$crumb"

	E2E_JENKINS_API_TOKEN=$(curl "$E2E_JENKINS_URL"/user/admin/descriptorByName/jenkins.security.ApiTokenProperty/generateNewToken \
	  --silent \
		--cookie cookies.txt \
		--header "$crumb" \
		--data "newTokenName=admin-token" \
		--user "$E2E_JENKINS_USER:$E2E_JENKINS_PASSWORD" | jq -r .data.tokenValue)
	export E2E_JENKINS_API_TOKEN
	rm -f cookies.txt
}

###############################################################################
# Creates global Git credentials for Jenkins.
# Globals:
#   exports E2E_JENKINS_GIT_CREDENTIAL_ID
###############################################################################
function create_jenkins_git_credentials() {
  E2E_JENKINS_GIT_CREDENTIAL_ID="lighthouse-e2e-tests-git-credentials"
  export E2E_JENKINS_GIT_CREDENTIAL_ID

	crumb=$(curl "$E2E_JENKINS_URL"/crumbIssuer/api/xml?xpath=concat\(//crumbRequestField,%22:%22,//crumb\) \
	  --silent --cookie-jar cookies.txt --user "$E2E_JENKINS_USER:$E2E_JENKINS_PASSWORD")

	echo "$crumb"

  json=$(cat <<EOF
json={
  "": "0",
  "credentials": {
    "scope": "GLOBAL",
    "id": "$E2E_JENKINS_GIT_CREDENTIAL_ID",
    "username": "$E2E_PRIMARY_SCM_USER",
    "password": "$E2E_PRIMARY_SCM_TOKEN",
    "description": "Git credentials for E2E Lighthouse tests",
    "stapler-class": "com.cloudbees.plugins.credentials.impl.UsernamePasswordCredentialsImpl",
    "$class": "com.cloudbees.plugins.credentials.impl.UsernamePasswordCredentialsImpl"
  }
}
EOF
)

	curl "$E2E_JENKINS_URL"/credentials/store/system/domain/_/createCredentials \
	  -X POST \
	  --silent \
		--cookie cookies.txt \
		--header "$crumb" \
		--user "$E2E_JENKINS_USER:$E2E_JENKINS_PASSWORD" \
    --data-urlencode "$json"

	rm -f cookies.txt
}

###############################################################################
# Installs Lighthouse with Jenkins as agent.
###############################################################################
function install_lighthouse() {
	# HMAC token is just a random 42 byte hex string we'll be using in Lighthouse and the webhook
	E2E_HMAC_TOKEN=$(LC_CTYPE=C tr -dc 'A-F0-9' </dev/urandom | head -c42)
	export E2E_HMAC_TOKEN

	gomplate -f bdd/jenkins/lighthouse-values.yaml.template -o lighthouse-values.yaml
	run_helm install -f lighthouse-values.yaml --namespace "$E2E_TEST_NAMESPACE" lighthouse charts/lighthouse
}

###############################################################################
# Runs E2E tests
###############################################################################
function run_e2e_tests() {
  make run-jenkins-e2e-tests
}

###############################################################################
# Main
#
# Parameters
#  $1 boolean flag determining whether test are running against existing/local cluster
###############################################################################
function main() {
  exe ensure_environment

  if ! $1; then
	  exe install_tools
	  exe create_cluster
	  exe configure_cluster
  fi

  exe install_jenkins
  exe create_jenkins_api_token
	exe create_jenkins_git_credentials
	exe install_lighthouse

	if ! make run-jenkins-e2e-tests; then
	  # If the tests fail, we remove the cluster cleanup trap to prevent garbage collection of the cluster
	  trap - EXIT
	fi
}

local_exec=false
while getopts ":l" opt; do
  case ${opt} in
    l )
      local_exec=true
      ;;
    \? )
      usage
      exit 1
      ;;
    : )
      usage
      exit 1
      ;;
  esac
done
shift $((OPTIND -1))

main $local_exec
