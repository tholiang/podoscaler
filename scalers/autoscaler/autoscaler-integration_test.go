package main

import (
	"testing"
)

func IntegrationMakeAutoscaler(node_avail_threshold float64, downscale_threshold float64, namespace string, maps int64, latency_threshold int64, metrics AutoscalerMetrics) Autoscaler {
	return Autoscaler{
		prometheus_url:                   "prometheus.url",
		min_node_availabiility_threshold: node_avail_threshold,
		downscale_utilization_threshold:  downscale_threshold,
		deployment_namespace:             namespace,
		maps:                             maps,
		latency_threshold:                latency_threshold,
		metrics:                          metrics,
	}
}

/* INTEGRATION TESTS TO BE RUN IN A CLUSTER */
/* PATCH UTILIZATION AND LATENCY */
func TestIntegration_BasicVscale(t *testing.T) {

}
