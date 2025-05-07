#!/bin/bash

# label the hotel res deployments with the vector label
deployments=("frontend-hotelres" "geo-hotelres" "profile-hotelres" "rate-hotelres" "recommendation-hotelres" "reservation-hotelres" "search-hotelres" "user-hotelres")

namespace="deathstarbench"

label_key="vecter"
label_value="true"

kubectl scale deployment -n deathstarbench --all --replicas=1

for deployment in "${deployments[@]}"; do
  echo "Labeling and setting resources for $deployment..."
  kubectl label deployment "$deployment" "$label_key=$label_value" -n "$namespace" --overwrite
  kubectl patch deployment $deployment -n deathstarbench --type="json" -p="[{'op': 'replace', 'path': '/spec/template/spec/containers/0/resources', 'value': {'requests': {'cpu': '300m'}}}]"
done

# build the autoscaler image, tag it, and push to dockerhub
docker image build -t watcher-img --build-arg BUILD_TAG=watcher ./scalers
docker tag watcher-img kvnz/watcher-img
docker push kvnz/watcher-img

# deploy the watcher
kubectl apply -f ./deploy/deploy-watcher.yaml

# deploy the autoscaler
for deployment in "${deployments[@]}"; do
  echo "Autoscaling $deployment..."
  kubectl autoscale deployment $deployment -n deathstarbench --cpu-percent=90 --min=1
done