#!/bin/bash
set -e
kubectl delete deployment testapp
kubectl delete svc testapp