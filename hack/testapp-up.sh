#!/bin/bash
set -e
eval $(minikube -p minikube docker-env)
docker image build -t testapp-img ./testapp
kubectl apply -f ./deploy/deploy-testapp.yaml
kubectl apply -f ./deploy/deploy-ingress.yaml
kubectl expose deployment/testapp --type="NodePort" --port 3000
sleep 3
kubectl port-forward svc/testapp 3000