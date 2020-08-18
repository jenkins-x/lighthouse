#!/usr/bin/env bash
set -e
set -x

mkdir -p extra-logs
kubectl logs --tail=-1 "$(kubectl get pod -l app=lighthouse-keeper -o jsonpath='{.items[*].metadata.name}')" > extra-logs/keeper.log
kubectl logs --tail=-1 "$(kubectl get pod -l app=lighthouse-foghorn -o jsonpath='{.items[*].metadata.name}')" > extra-logs/foghorn.log

# Capture the appropriate optional controllers
for controller in "$@"; do
  kubectl logs --tail=-1 "$(kubectl get pod -l "app=lighthouse-${controller}-controller" -o jsonpath='{.items[*].metadata.name}')" > "extra-logs/${controller}-controller.log"
done
lh_cnt=0
for lh_pod in $(kubectl get pod -l app=lighthouse-webhooks -o jsonpath='{.items[*].metadata.name}'); do
  ((lh_cnt=lh_cnt+1))
  kubectl logs --tail=-1 "${lh_pod}" > extra-logs/lh.${lh_cnt}.log
done

jx step stash -c lighthouse-tests -p "extra-logs/*.log" --bucket-url gs://jx-prod-logs
