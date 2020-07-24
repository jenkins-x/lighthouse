#!/usr/bin/env bash
set -e
set -x

# Activate the service account json provided by the pipeline
gcloud auth activate-service-account --key-file $GKE_SA

if [ -z "${USER}" ]; then
  USER=$(id -u -n)
fi

if [ -z "${BRANCH_NAME}" ]; then
  BRANCH_NAME=local
fi

if [ -z "${BUILD_ID}" ]; then
  BUILD_ID=$(tr -dc '0-9' < /dev/urandom | head -c5)
fi

CLUSTER_NAME=$( echo ${BRANCH_NAME}-lh-tekton-e2e-${BUILD_ID} | tr '[:upper:]' '[:lower:]' )
# Create the cluster with some standard labels and info for cleanup. Minimum version is 1.16.x
gcloud container clusters create ${CLUSTER_NAME} --num-nodes=3 --machine-type n1-standard-2 --enable-autoscaling --min-nodes=3 --max-nodes=5 --zone=europe-west1-c --scopes=https://www.googleapis.com/auth/cloud-platform,https://www.googleapis.com/auth/compute,https://www.googleapis.com/auth/devstorage.full_control,https://www.googleapis.com/auth/service.management,https://www.googleapis.com/auth/servicecontrol,https://www.googleapis.com/auth/logging.write,https://www.googleapis.com/auth/monitoring --labels=created-by=${USER},create-time=$(date "+%a-%b-%d-%Y-%H-%M-%S" |tr '[:upper:]' '[:lower:]'),cluster=lh-tekton-e2e,branch=$(echo $BRANCH_NAME | tr '[:upper:]' '[:lower:]') --project=jenkins-x-bdd3 --cluster-version=1.16.9-gke.6

# Install the nginx ingress controller
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v0.34.1/deploy/static/provider/cloud/deploy.yaml


# TODO: Replace this with something smarter
# Capture the external IP since it'll end up getting used in our "domain name", but loop for up to 2 minutes until we get it
iters=0
while [ $iters -lt 12 ]; do
  EXTERNAL_IP=$(kubectl get svc -n ingress-nginx ingress-nginx-controller -o jsonpath={.status.loadBalancer.ingress[0].ip})
  if [ -n "${EXTERNAL_IP}" ]; then
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

# HMAC token is just a random 42 byte hex string we'll be using in Lighthosue and the webhook
export E2E_HMAC_TOKEN=$(tr -dc 'A-F0-9' < /dev/urandom | head -c42)

# Take the template for values and replace the various placeholders - this could probably be cleaner.
cat bdd/tekton/values.yaml.template | sed 's/$VERSION/'"$VERSION"'/' | sed 's/$BOTUSER/'"$E2E_GIT_USER"'/' | sed 's/$HMACTOKEN/'"$E2E_HMAC_TOKEN"'/' | sed 's/$BOTSECRET/'"$E2E_PRIMARY_SCM_TOKEN"'/' | sed 's/$DOMAINNAME/'"$EXTERNAL_IP"'/' > myvalues.yaml

# helm 3 is installed on jx builders as "helm3", and that's what we want to use to install here.
# TODO: --validate=false is due to 'failed calling webhook "validate.nginx.ingress.kubernetes.io": ...' which I don't feel like digging into right now.
helm3 install -f myvalues.yaml --namespace lh-test lighthouse charts/lighthouse --validate=false

# set some other variables we're going to need in the e2e tests.
export E2E_GIT_SERVER=https://github.com
export E2E_GIT_KIND=github

# Run the test - we probably want something here to capture controller logs but we'll deal with that in a bit.
make run-e2e-tests

# Mark the cluster to be GC'd if we got this far and the tests passed
gcloud container clusters update ${CLUSTER_NAME} --project=jenkins-x-bdd3 --zone=europe-west1-c --update-labels=delete-me=true
