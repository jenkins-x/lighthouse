#!/usr/bin/env bash
set -e
set -x

export GL_USERNAME="jenkins-x-bot-test"
export GL_OWNER="jxbdd"
export GL_EMAIL="jenkins-x@googlegroups.com"

# fix broken `BUILD_NUMBER` env var
export BUILD_NUMBER="$BUILD_ID"

JX_HOME="/tmp/jxhome"
KUBECONFIG="/tmp/jxhome/config"

# lets avoid the git/credentials causing confusion during the test
export XDG_CONFIG_HOME=$JX_HOME

mkdir -p $JX_HOME/git

jx --version

# replace the credentials file with a single user entry
echo "https://$GL_USERNAME:$GL_ACCESS_TOKEN@gitlab.com" > $JX_HOME/git/credentials

gcloud auth activate-service-account --key-file $GKE_SA

# lets setup git 
git config --global --add user.name JenkinsXBot
git config --global --add user.email jenkins-x@googlegroups.com

echo "running the BDD tests with JX_HOME = $JX_HOME"

# setup jx boot parameters
export JX_VALUE_ADMINUSER_PASSWORD="$JENKINS_PASSWORD" # pragma: allowlist secret
export JX_VALUE_PIPELINEUSER_USERNAME="$GL_USERNAME"
export JX_VALUE_PIPELINEUSER_EMAIL="$GL_EMAIL"
export JX_VALUE_PIPELINEUSER_TOKEN="$GL_ACCESS_TOKEN"
export JX_VALUE_PROW_HMACTOKEN="$GL_ACCESS_TOKEN"

# TODO temporary hack until the batch mode in jx is fixed...
export JX_BATCH_MODE="true"

# Push the snapshot chart
pushd charts/lighthouse
make snapshot
popd

# TODO: Figure out why chatops doesn't work for gitlab!
export JX_ENABLE_TEST_CHATOPS_COMMANDS="false"

# Use the latest boot config promoted in the version stream instead of master to avoid conflicts during boot, because
# boot fetches always the latest version available in the version stream.
git clone  https://github.com/jenkins-x/jenkins-x-versions.git versions
export BOOT_CONFIG_VERSION=$(jx step get dependency-version --host=github.com --owner=jenkins-x --repo=jenkins-x-boot-config --dir versions | sed 's/.*: \(.*\)/\1/')
git clone https://github.com/jenkins-x/jenkins-x-boot-config.git boot-source
cd boot-source
git checkout tags/v${BOOT_CONFIG_VERSION} -b latest-boot-config

cp ../bdd/gitlab/jx-requirements.yml .
cp ../bdd/gitlab/parameters.yaml env
cp env/jenkins-x-platform/values.tmpl.yaml tmp.yaml
cat tmp.yaml ../bdd/boot-vault.platform.yaml > env/jenkins-x-platform/values.tmpl.yaml
rm tmp.yaml

# Manually interpolate lighthouse version tag
cat ../bdd/values.yaml.template >> env/lighthouse/values.tmpl.yaml
cp env/lighthouse/values.tmpl.yaml values.tmpl.yaml.tmp
sed 's/$VERSION/'"$VERSION"'/' values.tmpl.yaml.tmp > env/lighthouse/values.tmpl.yaml
cat env/lighthouse/values.tmpl.yaml
rm values.tmpl.yaml.tmp
sed -e s/\$VERSION/${VERSION}/g ../bdd/helm-requirements.yaml.template > env/requirements.yaml

echo "Building lighthouse with version $VERSION"

# TODO hack until we fix boot to do this too!
helm init --client-only
helm repo add jenkins-x https://storage.googleapis.com/chartmuseum.jenkins-x.io

set +e
jx step bdd \
    --versions-repo https://github.com/jenkins-x/jenkins-x-versions.git \
    --config ../bdd/gitlab/cluster.yaml \
    --gopath /tmp \
    --git-provider gitlab \
    --git-provider-url https://gitlab.com \
    --git-owner $GL_OWNER \
    --git-username $GL_USERNAME \
    --git-api-token $GL_ACCESS_TOKEN \
    --default-admin-password $JENKINS_PASSWORD \
    --no-delete-app \
    --no-delete-repo \
    --tests install \
    --tests test-create-spring

# Gitlab labels on pull requests aren't properly implemented yet on our side, so no quickstart tests - they depend on them.

bdd_result=$?
if [[ $bdd_result != 0 ]]; then
  mkdir -p extra-logs
  kubectl logs --tail=-1 "$(kubectl get pod -l app=controllerbuild -o jsonpath='{.items[*].metadata.name}')" > extra-logs/controllerbuild.log
  kubectl logs --tail=-1 "$(kubectl get pod -l app=tide -o jsonpath='{.items[*].metadata.name}')" > extra-logs/tide.log
  lh_cnt=0
  for lh_pod in $(kubectl get pod -l app=controllerbuild -o jsonpath='{.items[*].metadata.name}'); do
    ((lh_cnt=lh_cnt+1))
    kubectl logs --tail=-1 "${lh_pod}" > extra-logs/lh.${lh_cnt}.log
  done

  jx step stash -c lighthouse-tests -p extra-logs/*.log --bucket-url gs://jx-prod-logs
fi
cd ../charts/lighthouse
make delete-from-chartmuseum

exit $bdd_result
