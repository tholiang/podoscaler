#!/bin/bash
eval $(minikube -p minikube docker-env)
docker image build -t autoscaler-img --build-arg SRC_DIR=./autoscaler ./scalers
kubectl apply -f ./deploy/deploy-autoscaler.yaml
