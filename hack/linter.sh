#!/bin/bash

set -e -o pipefail

if [ "$DISABLE_LINTER" == "true" ]
then
  exit 0
fi

if ! command -v golangci-lint >/dev/null 2>&1; then
  echo "golangci-lint v2.12.2 is required but was not found on PATH."
  echo "Please install golangci-lint v2.12.2 (e.g. update your CI image or local install) and re-run."
  exit 1
fi

linterVersion="$(golangci-lint --version | awk '{print $4}')"

if [[ ! "${linterVersion}" =~ ^2\.12\.2 ]]; then
  echo "golangci-lint v2.12.2 is required (found: ${linterVersion:-none})."
  echo "Please install golangci-lint v2.12.2 (e.g. update your CI image or local install) and re-run."
  exit 1
fi

export GO111MODULE=on
golangci-lint run \
  --verbose \
  --build-tags build
