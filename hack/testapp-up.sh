#!/bin/bash
set -e
eval $(minikube -p minikube docker-env)
docker image build -t testapp-img ./testapp
kubectl apply -f ./deploy/deploy-testapp.yaml
kubectl apply -f ./deploy/deploy-ingress.yaml
