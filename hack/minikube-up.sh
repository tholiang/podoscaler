#!/bin/bash
minikube start --driver=docker --feature-gates=InPlacePodVerticalScaling=true
kubectl apply -f ./deploy/rbac.yaml
kubectl apply -f ./deploy/components.yaml