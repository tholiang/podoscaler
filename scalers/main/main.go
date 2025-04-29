package main

import (
	"time"

	autoscaler "github.com/tholiang/podoscaler/scalers/autoscaler"
	test "github.com/tholiang/podoscaler/scalers/autoscalertest"
)

func run_autoscaler() {
	am := new(autoscaler.DefaultAutoscalerMetrics)

	a := autoscaler.Autoscaler{
		PrometheusUrl:                 autoscaler.DEFAULT_PROMETHEUS_URL,
		MinNodeAvailabilityThreshold:  autoscaler.DEFAULT_MIN_NODE_AVAILABILITY_THRESHOLD,
		DownscaleUtilizationThreshold: autoscaler.DEFAULT_DOWNSCALE_UTILIZATION_THRESHOLD,
		DeploymentNamespace:           autoscaler.DEFAULT_DEPLOYMENT_NAMESPACE,
		Maps:                          autoscaler.DEFAULT_MAPS,
		LatencyThreshold:              autoscaler.DEFAULT_LATENCY_THRESHOLD,
		Metrics:                       am,
	}
	err := a.Init()
	if err != nil {
		panic(err)
	}

	for {
		err := a.RunRound()
		if err != nil {
			panic(err)
		}

		time.Sleep(5 * time.Second)
	}
}

func main() {
	// run_autoscaler()
	test.RunIntegrationTests()
}
