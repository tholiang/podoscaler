#!/bin/bash
set -e
eval $(minikube -p minikube docker-env)

echo "<<< BUILDING DUMMY APP >>>"
docker image build -q -t dummy-img ./dummy
kubectl apply -f ./deploy/deploy-dummy.yaml

echo "<<< BUILDING AUTOSCALER >>>"
docker image build -q -t autoscaler-img --build-arg BUILD_TAG=autoscalertest ./scalers
kubectl apply -f ./deploy/deploy-autoscaler-test.yaml

echo
echo "<<< RUNNING INTEGRATION TESTS >>>"

sleep 15

kubectl logs autoscaler-test

echo
kubectl delete deployment dummy
kubectl delete pod autoscaler-test