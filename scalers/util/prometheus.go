package util

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// return map of endpoint path to percentile latency
func GetLatencyMetrics(service_name string, percentile float64) (map[string]float64, error) {
	prom_url := os.Getenv("PROMETHEUS_URL")
	if prom_url == "" {
		return nil, errors.New("PROMETHEUS_URL env not set")
	}

	client, err := api.NewClient(api.Config{Address: prom_url})
	if err != nil {
		return nil, fmt.Errorf("Error creating client: %v", err)
	}

	v1api := v1.NewAPI(client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, warnings, err := v1api.Query(ctx, fmt.Sprintf("histogram_quantile(%f,rate(http_response_time_milliseconds_bucket{}[10m]))", percentile), time.Now())
	if err != nil {
		return nil, fmt.Errorf("Error querying prometheus: %v", err)
	}
	if len(warnings) > 0 {
		log.Printf("Warnings: %v", warnings)
	}

	metric_map := make(map[string]float64)
	if result.Type() == model.ValVector {
		vec := result.(model.Vector)
		if len(vec) == 0 {
			return nil, errors.New("No results returned")
		}
		for _, sample := range vec {
			sample_map := metricToMap(sample.Metric)
			if sample_map["service"] == service_name && sample_map["path"] != "" {
				metric_map[sample_map["path"]] = float64(sample.Value)
			}
		}
	} else {
		return nil, errors.New("Wrong result type")
	}

	return metric_map, nil
}

func metricToMap(m model.Metric) map[string]string {
	result := make(map[string]string)
	for name, value := range m {
		result[string(name)] = string(value)
	}
	return result
}
