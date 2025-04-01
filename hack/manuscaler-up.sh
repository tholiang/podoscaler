#!/bin/bash
eval $(minikube -p minikube docker-env)
docker image build -t manuscaler-img --build-arg SRC_DIR=./manuscaler ./scalers
kubectl apply -f ./deploy/deploy-manuscaler.yaml
kubectl expose deployment/manuscaler --type="NodePort" --port 3001
sleep 3
kubectl port-forward svc/manuscaler 3001
