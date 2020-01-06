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

git clone https://github.com/jenkins-x/jenkins-x-boot-config.git boot-source
cp bdd/gitlab/jx-requirements.yml boot-source
cp bdd/gitlab/parameters.yaml boot-source/env
cd boot-source

# Manually interpolate lighthouse version tag
cat ../bdd/values.yaml.template >> env/lighthouse/values.tmpl.yaml
cp env/lighthouse/values.tmpl.yaml values.tmpl.yaml.tmp
sed 's/$VERSION/'"$VERSION"'/' values.tmpl.yaml.tmp > env/lighthouse/values.tmpl.yaml
cat env/lighthouse/values.tmpl.yaml
rm values.tmpl.yaml.tmp

echo "Building lighthouse with version $VERSION"

# TODO hack until we fix boot to do this too!
helm init --client-only
helm repo add jenkins-x https://storage.googleapis.com/chartmuseum.jenkins-x.io


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
