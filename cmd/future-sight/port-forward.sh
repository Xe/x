#!/usr/bin/env bash

kubectl apply -f manifest.dev.yaml

kubectl port-forward -n future-sight svc/nats 4222:4222 &
kubectl port-forward -n future-sight deploy/minio 9000:9000 &
kubectl port-forward -n future-sight deploy/minio 9001:9001 &
kubectl port-forward -n future-sight svc/valkey 6379:6379 &

wait