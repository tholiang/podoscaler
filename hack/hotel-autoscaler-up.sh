#!/bin/bash

# label the hotel res deployments with the vector label
deployments=("frontend-hotelres" "geo-hotelres" "profile-hotelres" "rate-hotelres" "recommendation-hotelres" "reservation-hotelres" "search-hotelres" "user-hotelres")

namespace="deathstarbench"

label_key="vecter"
label_value="true"

for deployment in "${deployments[@]}"; do
  echo "Labeling $deployment..."
  kubectl label deployment "$deployment" "$label_key=$label_value" -n "$namespace" --overwrite
done

# build the autoscaler image, tag it, and push to dockerhub
docker image build -t autoscaler-img --build-arg BUILD_TAG=autoscaler ./scalers
docker tag autoscaler-img kvnz/autoscaler-img
docker push kvnz/autoscaler-img

# deploy the autoscaler
kubectl apply -f ./deploy/deploy-autoscaler.yaml