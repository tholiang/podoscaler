#!/bin/bash

set -e

# install gateway crds
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.2.0/standard-install.yaml

# install linkerd components
linkerd check --pre
linkerd install --crds | kubectl apply -f -
linkerd install | kubectl apply -f -
linkerd check
linkerd viz install -f linkerd-viz-values.yaml | kubectl apply -f -
linkerd check

# install metrics server
kubectl apply -f ~/setup/components.yaml
kubectl apply -f ~/setup/vecter/podoscaler/deploy/rbac.yaml