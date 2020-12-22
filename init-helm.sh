#!/bin/sh

helm init --client-only --skip-refresh
helm repo rm stable
helm repo add stable https://charts.helm.sh/stable
