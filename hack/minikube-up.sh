#!/bin/bash
set -e
minikube start --driver=docker --feature-gates=InPlacePodVerticalScaling=true
minikube addons enable ingress
linkerd install --crds | kubectl apply -f -
linkerd install --set proxyInit.runAsRoot=true | kubectl apply -f -
linkerd viz install | kubectl apply -f -
linkerd inject <(kubectl get deploy -n ingress-nginx ingress-nginx-controller -o yaml) | kubectl apply -f -
kubectl apply -f ./deploy/rbac.yaml
kubectl apply -f ./deploy/components.yaml