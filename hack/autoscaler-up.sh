#!/bin/bash
set -e
eval $(minikube -p minikube docker-env)
docker image build -t autoscaler-img --build-arg BUILD_TAG=autoscaler ./scalers
kubectl apply -f ./deploy/deploy-autoscaler.yaml
