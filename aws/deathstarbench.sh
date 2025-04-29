#!/bin/bash

set -e

# install deathstarbench hotelres
kubectl create namespace deathstarbench
helm install hotelres ~/DeathStar2Bench/hotelReservation/helm-chart/hotelreservation/ -n deathstarbench

# inject linkerd
kubectl get -n deathstarbench deploy -o yaml | linkerd inject - | kubectl apply -f -

# setup external frontend ip
. ~/setup/utils/edit_securitygroup_inbound_rules.sh
kubectl expose deployment frontend-hotelres -n deathstarbench --type=NodePort --name=frontend-service
bash ~/setup/utils/get_frontend_ip.sh -n deathstarbench
