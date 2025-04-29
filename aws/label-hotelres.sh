#!/bin/bash

deployments=("frontend-hotelres" "geo-hotelres" "profile-hotelres" "rate-hotelres" "recommendation-hotelres" "reservation-hotelres" "search-hotelres" "user-hotelres")

namespace="deathstarbench"

label_key="vecter"
label_value="true"

for deployment in "${deployments[@]}"; do
  echo "Labeling $deployment..."
  kubectl label deployment "$deployment" "$label_key=$label_value" -n "$namespace" --overwrite
done

echo "Labeling complete."
