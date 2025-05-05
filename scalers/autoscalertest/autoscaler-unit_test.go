//go:build autoscalertest
// +build autoscalertest

package autoscalertest

import (
	"testing"

	"github.com/tholiang/podoscaler/scalers/autoscaler"
)

func UnitMakeAutoscaler(node_avail_threshold float64, downscale_threshold float64, namespace string, Maps int64, LatencyThreshold int64, metrics autoscaler.AutoscalerMetrics) autoscaler.Autoscaler {
	return autoscaler.Autoscaler{
		PrometheusUrl:                 "prometheus.url",
		MinNodeAvailabilityThreshold:  node_avail_threshold,
		DownscaleUtilizationThreshold: downscale_threshold,
		Maps:                          Maps,
		LatencyThreshold:              LatencyThreshold,
		Metrics:                       metrics,
	}
}

/* FULL MOCK UNIT TESTS - DOESN'T CREATE ANY PODS (DOESN'T EVEN NEED TO BE RUN IN K8S) */
func TestUnit_BasicStable(t *testing.T) {
	// setup
	mm := CreateSimpleMockMetrics()

	// test
	a := UnitMakeAutoscaler(0.2, 0.85, MOCK_DEPLOYMENT_NAMESPACE, 500, 100, mm)
	err := a.Init()
	AssertNoError(err, t)

	err = a.RunRound()
	AssertNoError(err, t)

	AssertNoActions(mm, t)
}

func TestUnit_BasicVscaleUp(t *testing.T) {
	// values to test
	correctEndPods := map[string]PodData{
		"pod1": {PodName: "pod1", NodeName: "node1", ContainerName: "container", CpuRequests: 330},
		"pod2": {PodName: "pod2", NodeName: "node1", ContainerName: "container", CpuRequests: 330},
		"pod3": {PodName: "pod3", NodeName: "node2", ContainerName: "container", CpuRequests: 330},
	}

	// setup
	mm := CreateSimpleMockMetrics() // start 3 pods at 300 each
	mm.Latency = MOCK_LATENCY_THRESHOLD * 1.5
	mm.RelDeploymentUtil = 1.1
	mm.RelNodeUsages = map[string]float64{
		"node1": 0.9,
		"node2": 0.5,
	}

	// test
	a := UnitMakeAutoscaler(0.2, 0.85, MOCK_DEPLOYMENT_NAMESPACE, 400, 100, mm)
	err := a.Init()
	AssertNoError(err, t)

	err = a.RunRound()
	AssertNoError(err, t)

	AssertAction(mm, t, Action{Type: VscaleAction, PodName: "pod1", ContainerName: "container", CpuRequests: "330m"})
	AssertAction(mm, t, Action{Type: VscaleAction, PodName: "pod2", ContainerName: "container", CpuRequests: "330m"})
	AssertAction(mm, t, Action{Type: VscaleAction, PodName: "pod3", ContainerName: "container", CpuRequests: "330m"})
	AssertNoActions(mm, t)

	AssertPodListsEqual(mm.Pods, correctEndPods, t)
}

func TestUnit_BasicHscaleUp(t *testing.T) {
	// values to test
	correctEndPods := map[string]PodData{
		"pod1": {PodName: "pod1", NodeName: "node1", ContainerName: "container", CpuRequests: 450},
		"pod2": {PodName: "pod2", NodeName: "node1", ContainerName: "container", CpuRequests: 450},
		"pod3": {PodName: "pod3", NodeName: "node2", ContainerName: "container", CpuRequests: 450},
		"pod4": {PodName: "pod4", NodeName: "node2", ContainerName: "container", CpuRequests: 450},
	}

	// setup
	mm := CreateSimpleMockMetrics() // start 3 pods at 300 each
	mm.Latency = MOCK_LATENCY_THRESHOLD * 1.5
	mm.RelDeploymentUtil = 2
	mm.RelNodeUsages = map[string]float64{
		"node1": 0.9,
		"node2": 0.5,
	}

	// test
	a := UnitMakeAutoscaler(0.2, 0.85, MOCK_DEPLOYMENT_NAMESPACE, 500, 100, mm)
	err := a.Init()
	AssertNoError(err, t)

	err = a.RunRound()
	AssertNoError(err, t)

	AssertAction(mm, t, Action{Type: ChangeReplicaCountAction, Namespace: MOCK_DEPLOYMENT_NAMESPACE, DeploymentName: MOCK_DEPLOYMENT_NAME, ReplicaCt: 4})
	AssertAction(mm, t, Action{Type: VscaleAction, PodName: "pod1", ContainerName: "container", CpuRequests: "450m"})
	AssertAction(mm, t, Action{Type: VscaleAction, PodName: "pod2", ContainerName: "container", CpuRequests: "450m"})
	AssertAction(mm, t, Action{Type: VscaleAction, PodName: "pod3", ContainerName: "container", CpuRequests: "450m"})
	AssertAction(mm, t, Action{Type: VscaleAction, PodName: "pod4", ContainerName: "container", CpuRequests: "450m"})
	AssertNoActions(mm, t)

	AssertPodListsEqual(mm.Pods, correctEndPods, t)
}

func TestUnit_BasicVscaleDown(t *testing.T) {
	// values to test
	correctEndPods := map[string]PodData{
		"pod1": {PodName: "pod1", NodeName: "node1", ContainerName: "container", CpuRequests: 283}, // ceil(240 / 0.85) = 283
		"pod2": {PodName: "pod2", NodeName: "node1", ContainerName: "container", CpuRequests: 283},
		"pod3": {PodName: "pod3", NodeName: "node2", ContainerName: "container", CpuRequests: 283},
	}

	// setup
	mm := CreateSimpleMockMetrics() // start 3 pods at 300 each
	mm.Latency = MOCK_LATENCY_THRESHOLD * 0.9
	mm.RelDeploymentUtil = 0.8 // default alloc is 900

	// test
	a := UnitMakeAutoscaler(0.2, 0.85, MOCK_DEPLOYMENT_NAMESPACE, 300, 100, mm)
	err := a.Init()
	AssertNoError(err, t)

	err = a.RunRound()
	AssertNoError(err, t)

	AssertAction(mm, t, Action{Type: VscaleAction, PodName: "pod1", ContainerName: "container", CpuRequests: "283m"})
	AssertAction(mm, t, Action{Type: VscaleAction, PodName: "pod2", ContainerName: "container", CpuRequests: "283m"})
	AssertAction(mm, t, Action{Type: VscaleAction, PodName: "pod3", ContainerName: "container", CpuRequests: "283m"})
	AssertNoActions(mm, t)

	AssertPodListsEqual(mm.Pods, correctEndPods, t)
}

func TestUnit_BasicHscaleDown(t *testing.T) {
	// values to test
	correctEndPods := map[string]PodData{
		"pod1": {PodName: "pod1", NodeName: "node1", ContainerName: "container", CpuRequests: 265}, // ceil(225 / 0.85) = 265
		"pod2": {PodName: "pod2", NodeName: "node1", ContainerName: "container", CpuRequests: 265},
	}

	// setup
	mm := CreateSimpleMockMetrics() // start 3 pods at 300 each
	mm.Latency = MOCK_LATENCY_THRESHOLD * 0.9
	mm.RelDeploymentUtil = 0.5 // default alloc is 900

	// test
	a := UnitMakeAutoscaler(0.2, 0.85, MOCK_DEPLOYMENT_NAMESPACE, 300, 100, mm)
	err := a.Init()
	AssertNoError(err, t)

	err = a.RunRound()
	AssertNoError(err, t)

	AssertAction(mm, t, Action{Type: ChangeReplicaCountAction, Namespace: MOCK_DEPLOYMENT_NAMESPACE, DeploymentName: MOCK_DEPLOYMENT_NAME, ReplicaCt: 2})
	AssertAction(mm, t, Action{Type: VscaleAction, PodName: "pod1", ContainerName: "container", CpuRequests: "265m"})
	AssertAction(mm, t, Action{Type: VscaleAction, PodName: "pod2", ContainerName: "container", CpuRequests: "265m"})
	AssertNoActions(mm, t)

	AssertPodListsEqual(mm.Pods, correctEndPods, t)
}

func TestUnit_NoCongestion(t *testing.T) {
	// values to test
	correctEndPods := map[string]PodData{
		"pod1": {PodName: "pod1", NodeName: "node1", ContainerName: "container", CpuRequests: 300},
		"pod2": {PodName: "pod2", NodeName: "node1", ContainerName: "container", CpuRequests: 300},
		"pod3": {PodName: "pod3", NodeName: "node2", ContainerName: "container", CpuRequests: 300},
	}

	// setup
	mm := CreateSimpleMockMetrics() // start 3 pods at 300 each
	mm.Latency = MOCK_LATENCY_THRESHOLD * 1.5
	mm.RelDeploymentUtil = 0.9
	mm.RelNodeUsages = map[string]float64{
		"node1": 0.6,
		"node2": 0.3,
	}

	// test
	a := UnitMakeAutoscaler(0.2, 0.85, MOCK_DEPLOYMENT_NAMESPACE, 400, 100, mm)
	err := a.Init()
	AssertNoError(err, t)

	err = a.RunRound()
	AssertNoError(err, t)

	AssertNoActions(mm, t)

	AssertPodListsEqual(mm.Pods, correctEndPods, t)
}

// func TestUnit_PodMove(t *testing.T) {
// 	// values to test
// 	correctEndPods := map[string]PodData{
// 		"pod2": {PodName: "pod2", NodeName: "node1", ContainerName: "container", CpuRequests: 330},
// 		"pod3": {PodName: "pod3", NodeName: "node2", ContainerName: "container", CpuRequests: 330},
// 		"pod4": {PodName: "pod4", NodeName: "node2", ContainerName: "container", CpuRequests: 330},
// 	}

// 	// setup
// 	mm := CreateSimpleMockMetrics() // start 3 pods at 300 each
// 	mm.Latency = MOCK_LATENCY_THRESHOLD * 1.5
// 	mm.RelDeploymentUtil = 1.1
// 	mm.RelNodeUsages = map[string]float64{
// 		"node1": 1,
// 		"node2": 0.5,
// 	}
// 	mm.NodeAllocables = map[string]int64{
// 		"node1": 10,
// 		"node2": 700,
// 	}

// 	// test
// 	a := UnitMakeAutoscaler(0.2, 0.85, MOCK_DEPLOYMENT_NAMESPACE, 400, 100, mm)
// 	err := a.Init()
// 	AssertNoError(err, t)

// 	err = a.RunRound()
// 	AssertNoError(err, t)

// 	AssertAction(mm, t, Action{Type: ChangeReplicaCountAction, Namespace: MOCK_DEPLOYMENT_NAMESPACE, DeploymentName: MOCK_DEPLOYMENT_NAME, ReplicaCt: 4})
// 	AssertAction(mm, t, Action{Type: DeletePodAction, Namespace: MOCK_DEPLOYMENT_NAMESPACE, PodName: "pod1"}) // funky - can sometimes be pod2
// 	AssertAction(mm, t, Action{Type: ChangeReplicaCountAction, Namespace: MOCK_DEPLOYMENT_NAMESPACE, DeploymentName: MOCK_DEPLOYMENT_NAME, ReplicaCt: 3})
// 	AssertAction(mm, t, Action{Type: VscaleAction, PodName: "pod2", ContainerName: "container", CpuRequests: "330m"})
// 	AssertAction(mm, t, Action{Type: VscaleAction, PodName: "pod3", ContainerName: "container", CpuRequests: "330m"})
// 	AssertAction(mm, t, Action{Type: VscaleAction, PodName: "pod4", ContainerName: "container", CpuRequests: "330m"})
// 	AssertNoActions(mm, t)

// 	AssertPodListsEqual(mm.Pods, correctEndPods, t)
// }

// error handling?
