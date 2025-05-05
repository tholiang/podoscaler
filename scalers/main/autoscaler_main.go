//go:build autoscaler
// +build autoscaler

package main

import (
	"time"

	autoscaler "github.com/tholiang/podoscaler/scalers/autoscaler"
	"github.com/tholiang/podoscaler/scalers/util"
)

func run_autoscaler() {
	am := new(autoscaler.DefaultAutoscalerMetrics)

	a := autoscaler.Autoscaler{
		PrometheusUrl:                 util.DEFAULT_PROMETHEUS_URL,
		MinNodeAvailabilityThreshold:  autoscaler.DEFAULT_MIN_NODE_AVAILABILITY_THRESHOLD,
		DownscaleUtilizationThreshold: autoscaler.DEFAULT_DOWNSCALE_UTILIZATION_THRESHOLD,

		Maps:             autoscaler.DEFAULT_MAPS,
		LatencyThreshold: autoscaler.DEFAULT_LATENCY_THRESHOLD,
		Metrics:          am,
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

		time.Sleep(60 * time.Second)
	}
}

func main() {
	run_autoscaler()
}
