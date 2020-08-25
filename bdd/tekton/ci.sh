#!/usr/bin/env bash
set -e

# Error out if required environment variables aren't set.
missingEnvVarMessages=()
if [ -z "${VERSION}" ]; then
  missingEnvVarMessages+=( "VERSION for Lighthouse images not set" )
fi
if [ -z "${E2E_PRIMARY_SCM_USER}" ]; then
  missingEnvVarMessages+=( "E2E_PRIMARY_SCM_USER for bot git provider user not set" )
fi
if [ -z "${E2E_APPROVER_SCM_USER}" ]; then
  missingEnvVarMessages+=( "E2E_APPROVER_SCM_USER for approver git provider user not set" )
fi
if [ -z "${E2E_PRIMARY_SCM_TOKEN}" ]; then
  missingEnvVarMessages+=( "E2E_PRIMARY_SCM_TOKEN for bot git provider token not set" )
fi
if [ -z "${E2E_APPROVER_SCM_TOKEN}" ]; then
  missingEnvVarMessages+=( "E2E_APPROVER_SCM_TOKEN for approver git provider token not set" )
fi
if [ -z "${E2E_GIT_KIND}" ]; then
  missingEnvVarMessages+=( "E2E_GIT_KIND for git flavor (one of 'github', 'gitlab', 'stash') not set" )
fi
if [ -z "${E2E_GIT_SERVER}" ]; then
  missingEnvVarMessages+=( "E2E_GIT_SERVER for git server base URL (i.e., 'https://github.com') not set" )
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

CLUSTER_NAME=$( echo ${BRANCH_NAME,,}-${BUILD_ID,,}-lh-tekton-e2e )

dateLabel=$(date "+%a-%b-%d-%Y-%H-%M-%S")

# Update gcloud so we can do --release-channel
gcloud components update -q

# Activate the service account json provided by the pipeline, if it exists
if [ -n "${GKE_SA}" ]; then
  gcloud auth activate-service-account --key-file "${GKE_SA}"
else
  echo "GKE_SA environment variable not set, so using current GKE login to create cluster"
fi

# Create the cluster with some standard labels and info for cleanup. Minimum version is 1.16.x
gcloud container clusters create "${CLUSTER_NAME}" --num-nodes=3 --machine-type n1-standard-2 --enable-autoscaling --min-nodes=3 --max-nodes=5 --zone=europe-west1-c --scopes=https://www.googleapis.com/auth/cloud-platform,https://www.googleapis.com/auth/compute,https://www.googleapis.com/auth/devstorage.full_control,https://www.googleapis.com/auth/service.management,https://www.googleapis.com/auth/servicecontrol,https://www.googleapis.com/auth/logging.write,https://www.googleapis.com/auth/monitoring --labels="created-by=${USER},create-time=${dateLabel,,},cluster=lh-tekton-e2e,branch=${BRANCH_NAME,,}" --project=jenkins-x-bdd3 --release-channel=regular

# Install the nginx ingress controller
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v0.34.1/deploy/static/provider/cloud/deploy.yaml
# And delete the validating webhook config because it causes issues, since helm3 can't access the endpoint from outside the cluster
kubectl delete validatingwebhookconfigurations.admissionregistration.k8s.io ingress-nginx-admission

# TODO: Replace this with something smarter
# Capture the external IP since it'll end up getting used in our "domain name", but loop for up to 2 minutes until we get it
iters=0
while [ $iters -lt 12 ]; do
  capturedIp=$(kubectl get svc -n ingress-nginx ingress-nginx-controller -o jsonpath="{.status.loadBalancer.ingress[0].ip}")
  if [ -n "${capturedIp}" ]; then
    export EXTERNAL_IP=${capturedIp}
    break
  fi
  iters=$((iters + 1))
  sleep 10
done

# Create our test namespace and switch into it
kubectl create namespace lh-test
kubectl config set-context --current --namespace=lh-test

# Download the Tekton v0.14.2 release YAML, switch the namespace in it, and apply it.
curl https://storage.googleapis.com/tekton-releases/pipeline/previous/v0.14.2/release.yaml | sed -E "s/namespace\: tekton-pipelines/namespace: lh-test/" > install-tekton.yml
kubectl apply -f install-tekton.yml

# HMAC token is just a random 42 byte hex string we'll be using in Lighthouse and the webhook
E2E_HMAC_TOKEN=$(tr -dc 'A-F0-9' < /dev/urandom | head -c42)
export E2E_HMAC_TOKEN

# install gomplate for generating the myvalues.yaml from the template
pushd bdd/tekton
go get github.com/hairyhenderson/gomplate/v3/cmd/gomplate
popd

# Take the template for values and replace the various placeholders
gomplate -f bdd/tekton/values.yaml.template -o myvalues.yaml
#cat bdd/tekton/values.yaml.template | sed 's/$VERSION/'"$VERSION"'/' | sed 's/$BOTUSER/'"$E2E_PRIMARY_SCM_USER"'/' | sed 's/$HMACTOKEN/'"$E2E_HMAC_TOKEN"'/' | sed 's/$BOTSECRET/'"$E2E_PRIMARY_SCM_TOKEN"'/' | sed 's/$DOMAINNAME/'"$EXTERNAL_IP"'/' > myvalues.yaml

# helm 3 is installed on jx builders as "helm3", and that's what we want to use to install here.
helm3 install -f myvalues.yaml --namespace lh-test lighthouse charts/lighthouse

# Make sure we didn't create the config ConfigMap, since that should only be created if explicitly specified
set +e
if kubectl get configmap config ; then
  echo "Shouldn't have gotten the 'config' ConfigMap, but did"
  exit 1
fi

# Run the test - we probably want something here to capture controller logs but we'll deal with that in a bit.
make run-e2e-tests

bdd_result=$?
if [[ $bdd_result != 0 ]]; then
  bash bdd/capture-failed-pod-logs.sh tekton
else
  # Mark the cluster to be GC'd if we got this far and the tests passed
  gcloud container clusters update "${CLUSTER_NAME}" --project=jenkins-x-bdd3 --zone=europe-west1-c --update-labels=delete-me=true
fi
