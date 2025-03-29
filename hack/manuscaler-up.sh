#!/bin/bash
eval $(minikube -p minikube docker-env)
docker image build -t manuscaler-img ./manuscaler
kubectl apply -f ./manuscaler/rbac.yaml
kubectl apply -f ./manuscaler/deployment.yaml
kubectl expose deployment/manuscaler --type="NodePort" --port 3001
kubectl port-forward svc/manuscaler 3001
