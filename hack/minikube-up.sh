#!/bin/bash
set -e
minikube start --driver=docker --feature-gates=InPlacePodVerticalScaling=true
minikube addons enable ingress
kubectl apply -f ./deploy/rbac.yaml
kubectl apply -f ./deploy/components.yaml
