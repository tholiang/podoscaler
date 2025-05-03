#!/bin/bash

set -e

# install deathstarbench hotelres
kubectl create namespace deathstarbench
helm install hotelres ~/DeathStar2Bench/hotelReservation/helm-chart/hotelreservation/ -n deathstarbench
