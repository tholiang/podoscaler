#!/bin/bash
set -e
minikube start --nodes 2 --driver=docker --feature-gates=InPlacePodVerticalScaling=true
kubectl apply -f ./deploy/rbac.yaml
kubectl apply -f ./deploy/components.yaml