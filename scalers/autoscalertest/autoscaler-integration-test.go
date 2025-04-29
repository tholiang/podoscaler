package autoscalertest

import (
	"fmt"

	"github.com/tholiang/podoscaler/scalers/autoscaler"
)

func integration_make_autoscaler(node_avail_threshold float64, downscale_threshold float64, namespace string, Maps int64, LatencyThreshold int64, metrics autoscaler.AutoscalerMetrics) autoscaler.Autoscaler {
	return autoscaler.Autoscaler{
		PrometheusUrl:                 autoscaler.DEFAULT_PROMETHEUS_URL,
		MinNodeAvailabilityThreshold:  node_avail_threshold,
		DownscaleUtilizationThreshold: downscale_threshold,
		DeploymentNamespace:           namespace,
		Maps:                          Maps,
		LatencyThreshold:              LatencyThreshold,
		Metrics:                       metrics,
	}
}

/* INTEGRATION TESTS TO BE RUN IN A CLUSTER */
/* PATCH UTILIZATION AND LATENCY */
func IntegrationTest_BasicStable() {
	// setup
	mm := CreateIntMockMetrics()

	// test
	a := integration_make_autoscaler(0.2, 0.85, "default", 500, 100, mm)
	err := a.Init()
	IntAssertNoError(err)

	err = a.RunRound()
	IntAssertNoError(err)

	IntAssertNoActions(mm)
}

func RunIntegrationTests() {
	IntegrationTest_BasicStable()

	fmt.Println("<<< TESTS PASSED SUCCESSFULLY >>>")
}
