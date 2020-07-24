#!/bin/bash
set -e
set -x

gcloud container clusters create lh-tekton-e2e-${BUILD_ID} --num-nodes=3 --machine-type n1-standard-2 --enable-autoscaling --min-nodes=3 --max-nodes=5 --zone=europe-west1-c --scopes=https://www.googleapis.com/auth/cloud-platform,https://www.googleapis.com/auth/compute,https://www.googleapis.com/auth/devstorage.full_control,https://www.googleapis.com/auth/service.management,https://www.googleapis.com/auth/servicecontrol,https://www.googleapis.com/auth/logging.write,https://www.googleapis.com/auth/monitoring --labels=created-by=${USER},create-time=$(date "+%a-%b-%d-%Y-%H-%M-%S" |tr '[:upper:]' '[:lower:]'),cluster=lh-tekton-e2e,branch=$(echo $BRANCH | tr '[:upper:]' '[:lower:]')

kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v0.34.1/deploy/static/provider/cloud/deploy.yaml

# Sleep 30 seconds to wait for the ingress controller to get an external IP and then capture it

EXTERNAL_IP=$(kubectl get svc -n ingress-nginx ingress-nginx-controller -o jsonpath={.status.loadBalancer.ingress[0].ip})

kubectl create namespace lh-test

kubectl config set-context --current --namespace=lh-test

curl https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml | sed -E "s/namespace\: tekton-pipelines/namespace: lh-test/" > install-tekton.yml

kubectl apply -f install-tekton.yml

export E2E_HMAC_TOKEN=$(openssl rand -hex 21)

cat bdd/tekton/values.yaml.template | sed 's/$VERSION/'"$VERSION"'/' | sed 's/$BOTUSER/'"$E2E_GIT_USER"'/' | sed 's/$HMACTOKEN/'"$E2E_HMAC_TOKEN"'/' | sed 's/$BOTSECRET/'"$E2E_PRIMARY_SCM_TOKEN"'/' | sed 's/$DOMAINNAME/'"$EXTERNAL_IP"'/' > myvalues.yaml

helm3 install -f myvalues.yaml --namespace lh-test lighthouse charts/lighthouse

export E2E_GIT_SERVER=https://github.com
export E2E_GIT_KIND=github

make run-e2e-tests

