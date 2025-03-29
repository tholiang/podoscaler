#!/bin/bash
eval $(minikube -p minikube docker-env)
docker image build -t testapp-img ./testapp
kubectl apply -f ./testapp/deployment.yaml
