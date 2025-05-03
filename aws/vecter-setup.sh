#!/bin/bash

set -e

# cloudwatch secrets
kubectl create secret generic aws-secrets --from-literal=AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID --from-literal=AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY --from-literal=AWS_REGION=$AWS_REGION

# install metrics server
kubectl apply -f ~/setup/components.yaml
kubectl apply -f ~/setup/vecter/podoscaler/deploy/rbac.yaml
