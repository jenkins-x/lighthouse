#!/usr/bin/env bash

# Copyright 2017 The Kubernetes Authors.

set -o errexit
set -o nounset
set -o pipefail


# generate-groups generates everything for a project with external types only, e.g. a project based
# on CustomResourceDefinitions.

if [ "$#" -lt 4 ] || [ "${1}" == "--help" ]; then
  cat <<EOF
Usage: $(basename "$0") <generators> <output-package> <apis-package> <groups-versions> ...
  <generators>        the generators comma separated to run (deepcopy,defaulter,client,lister,informer) or "all".
  <output-package>    the output package name (e.g. github.com/example/project/pkg/generated).
  <apis-package>      the external types dir (e.g. github.com/example/api or github.com/example/project/pkg/apis).
  <groups-versions>   the groups and their versions in the format "groupA:v1,v2 groupB:v1 groupC:v2", relative
                      to <api-package>.
  ...                 arbitrary flags passed to all generator binaries.
Examples:
  $(basename "$0") all             github.com/example/project/pkg/client github.com/example/project/pkg/apis "foo:v1 bar:v1alpha1,v1beta1"
  $(basename "$0") deepcopy,client github.com/example/project/pkg/client github.com/example/project/pkg/apis "foo:v1 bar:v1alpha1,v1beta1"
EOF
  exit 0
fi

GENS="$1"
OUTPUT_PKG="$2"
APIS_PKG="$3"
GROUPS_WITH_VERSIONS="$4"
shift 4

GENERATOR_VERSION=v0.17.6
(
  # To support running this script from anywhere, we have to first cd into this directory
  # so we can install the tools.
  cd "$(dirname "${0}")"
  go get k8s.io/code-generator/cmd/{defaulter-gen,client-gen,lister-gen,informer-gen,deepcopy-gen}@$GENERATOR_VERSION
)

PREFIX=${GOBIN:-${GOPATH}/bin}

# turn off module mode before running the generators
# https://github.com/kubernetes/code-generator/issues/69
# we also need to populate vendor
readonly REPO_ROOT_DIR="${REPO_ROOT_DIR:-$(git rev-parse --show-toplevel 2> /dev/null)}"

go mod vendor

export GO111MODULE="off"

# fake being in a gopath
FAKE_GOPATH="$(mktemp -d)"
trap 'rm -rf ${FAKE_GOPATH} && rm -rf ${REPO_ROOT_DIR}/vendor' EXIT

FAKE_REPOPATH="${FAKE_GOPATH}/src/github.com/jenkins-x/lighthouse"
mkdir -p "$(dirname "${FAKE_REPOPATH}")" && ln -s "${REPO_ROOT_DIR}" "${FAKE_REPOPATH}"

export GOPATH="${FAKE_GOPATH}"
cd "${FAKE_REPOPATH}"

function codegen::join() { local IFS="$1"; shift; echo "$*"; }

# enumerate group versions
FQ_APIS=() # e.g. k8s.io/api/apps/v1
for GVs in ${GROUPS_WITH_VERSIONS}; do
  IFS=: read -r G Vs <<<"${GVs}"

  # enumerate versions
  for V in ${Vs//,/ }; do
    FQ_APIS+=("${APIS_PKG}/${G}/${V}")
  done
done

# All generators expect a mandatory boilerplate file as input. Since we don't add any boilerplate we use process substitution '<( )' to pass a dummy empty file.
# See also https://github.com/kubernetes/code-generator/issues/6
if [ "${GENS}" = "all" ] || grep -qw "deepcopy" <<<"${GENS}"; then
  echo "Generating deepcopy funcs for ${GROUPS_WITH_VERSIONS}"
  "${PREFIX}/deepcopy-gen" --input-dirs "$(codegen::join , "${FQ_APIS[@]}")" -O zz_generated.deepcopy --bounding-dirs "${APIS_PKG}" --go-header-file <(echo "")
fi

if [ "${GENS}" = "all" ] || grep -qw "client" <<<"${GENS}"; then
  echo "Generating clientset for ${GROUPS_WITH_VERSIONS} at ${OUTPUT_PKG}/${CLIENTSET_PKG_NAME:-clientset}"
  "${PREFIX}/client-gen" --clientset-name "${CLIENTSET_NAME_VERSIONED:-versioned}" --input-base "" --input "$(codegen::join , "${FQ_APIS[@]}")" --output-package "${OUTPUT_PKG}/${CLIENTSET_PKG_NAME:-clientset}" --go-header-file <(echo "")
fi

if [ "${GENS}" = "all" ] || grep -qw "lister" <<<"${GENS}"; then
  echo "Generating listers for ${GROUPS_WITH_VERSIONS} at ${OUTPUT_PKG}/listers"
  "${PREFIX}/lister-gen" --input-dirs "$(codegen::join , "${FQ_APIS[@]}")" --output-package "${OUTPUT_PKG}/listers" --go-header-file <(echo "")
fi

if [ "${GENS}" = "all" ] || grep -qw "informer" <<<"${GENS}"; then
  echo "Generating informers for ${GROUPS_WITH_VERSIONS} at ${OUTPUT_PKG}/informers"
  "${PREFIX}/informer-gen" \
           --input-dirs "$(codegen::join , "${FQ_APIS[@]}")" \
           --versioned-clientset-package "${OUTPUT_PKG}/${CLIENTSET_PKG_NAME:-clientset}/${CLIENTSET_NAME_VERSIONED:-versioned}" \
           --listers-package "${OUTPUT_PKG}/listers" \
           --output-package "${OUTPUT_PKG}/informers" \
           --go-header-file <(echo "")
fi

export GO111MODULE="on"
cd $REPO_ROOT_DIR
