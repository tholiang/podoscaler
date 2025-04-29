#!/bin/bash
set -e
eval $(minikube -p minikube docker-env)
kubectl delete deployment dummy
kubectl delete deployment autoscaler

docker image build -t dummy-img ./dummy
kubectl apply -f ./deploy/deploy-dummy.yaml

docker image build -t autoscaler-img --build-arg SRC_DIR=./main ./scalers
kubectl apply -f ./deploy/deploy-autoscaler.yaml