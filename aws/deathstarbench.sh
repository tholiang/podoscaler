#!/bin/bash

set -e

# install deathstarbench hotelres
kubectl create namespace deathstarbench
helm install hotelres ~/DeathStar2Bench/hotelReservation/helm-chart/hotelreservation/ -n deathstarbench

# inject linkerd
kubectl get -n deathstarbench deploy -o yaml | linkerd inject - | kubectl apply -f -

# setup frontend-service
kubectl expose deployment frontend-hotelres -n deathstarbench --type=NodePort --name=frontend-service
