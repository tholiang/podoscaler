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
  kubectl patch deployment $deployment -n deathstarbench --type="json" -p="[{'op': 'replace', 'path': '/spec/template/spec/containers/0/resources', 'value': {'requests': {'cpu': '100m'}}}]"
done

# build the autoscaler image, tag it, and push to dockerhub
docker image build -t autoscaler-img --build-arg BUILD_TAG=autoscaler ./scalers
docker tag autoscaler-img kvnz/autoscaler-img
docker push kvnz/autoscaler-img

# build the autoscaler image, tag it, and push to dockerhub
docker image build -t watcher-img --build-arg BUILD_TAG=watcher ./scalers
docker tag watcher-img kvnz/watcher-img
docker push kvnz/watcher-img

# deploy the autoscaler
kubectl apply -f ./deploy/deploy-autoscaler.yaml
kubectl apply -f ./deploy/deploy-watcher.yaml