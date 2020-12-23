#!/bin/sh

helm init --client-only --skip-refresh --stable-repo-url https://charts.helm.sh/stable
#helm repo rm stable
#helm repo add stable https://charts.helm.sh/stable
