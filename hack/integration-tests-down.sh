#!/bin/bash
set -e
kubectl delete deployment dummy
kubectl delete pod autoscaler-test