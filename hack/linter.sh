#!/bin/bash

set -e -o pipefail

if [ "$DISABLE_LINTER" == "true" ]
then
  exit 0
fi

linterVersion="$(golangci-lint --version | awk '{print $4}')"

if [[ ! "${linterVersion}" =~ ^2\.12\.2 ]]; then
	echo "Installing golangci-lint"
  exit 1
fi

export GO111MODULE=on
golangci-lint run \
  --verbose \
  --build-tags build
