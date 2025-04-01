#!/bin/bash
eval $(minikube -p minikube docker-env)
docker image build -t autoscaler-img ./autoscaler
kubectl apply -f ./autoscaler/rbac.yaml
kubectl apply -f ./autoscaler/deployment.yaml
