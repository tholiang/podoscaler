package main

import (
	"testing"

	testutil "github.com/tholiang/podoscaler/scalers/testutil"
)

func UnitMakeAutoscaler(node_avail_threshold float64, downscale_threshold float64, namespace string, maps int64, latency_threshold int64, metrics AutoscalerMetrics) Autoscaler {
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

/* FULL MOCK UNIT TESTS - DOESN'T CREATE ANY PODS (DOESN'T EVEN NEED TO BE RUN IN K8S) */
func TestUnit_BasicStable(t *testing.T) {
	// setup
	mm := testutil.CreateSimpleMockMetrics()

	// test
	a := UnitMakeAutoscaler(0.2, 0.85, testutil.MOCK_DEPLOYMENT_NAMESPACE, 500, 100, mm)
	err := a.Init()
	testutil.AssertNoError(err, t)

	err = a.RunRound()
	testutil.AssertNoError(err, t)

	testutil.AssertNoActions(mm, t)
}

func TestUnit_BasicVscaleUp(t *testing.T) {
	// values to test
	correctEndPods := map[string]testutil.PodData{
		"pod1": {PodName: "pod1", NodeName: "node1", ContainerName: "container", CpuRequests: 330},
		"pod2": {PodName: "pod2", NodeName: "node1", ContainerName: "container", CpuRequests: 330},
		"pod3": {PodName: "pod3", NodeName: "node2", ContainerName: "container", CpuRequests: 330},
	}

	// setup
	mm := testutil.CreateSimpleMockMetrics() // start 3 pods at 300 each
	mm.Latency = testutil.MOCK_LATENCY_THRESHOLD * 1.5
	mm.RelDeploymentUtil = 1.1
	mm.NodeUsages = map[string]int64{
		"node1": 900,
		"node2": 500,
	}

	// test
	a := UnitMakeAutoscaler(0.2, 0.85, testutil.MOCK_DEPLOYMENT_NAMESPACE, 400, 100, mm)
	err := a.Init()
	testutil.AssertNoError(err, t)

	err = a.RunRound()
	testutil.AssertNoError(err, t)

	testutil.AssertAction(mm, t, testutil.Action{Type: testutil.VscaleAction, PodName: "pod1", ContainerName: "container", CpuRequests: "330m"})
	testutil.AssertAction(mm, t, testutil.Action{Type: testutil.VscaleAction, PodName: "pod2", ContainerName: "container", CpuRequests: "330m"})
	testutil.AssertAction(mm, t, testutil.Action{Type: testutil.VscaleAction, PodName: "pod3", ContainerName: "container", CpuRequests: "330m"})
	testutil.AssertNoActions(mm, t)

	testutil.AssertPodListsEqual(mm.Pods, correctEndPods, t)
}

func TestUnit_BasicHscaleUp(t *testing.T) {
	// values to test
	correctEndPods := map[string]testutil.PodData{
		"pod1": {PodName: "pod1", NodeName: "node1", ContainerName: "container", CpuRequests: 450},
		"pod2": {PodName: "pod2", NodeName: "node1", ContainerName: "container", CpuRequests: 450},
		"pod3": {PodName: "pod3", NodeName: "node2", ContainerName: "container", CpuRequests: 450},
		"pod4": {PodName: "pod4", NodeName: "node2", ContainerName: "container", CpuRequests: 450},
	}

	// setup
	mm := testutil.CreateSimpleMockMetrics() // start 3 pods at 300 each
	mm.Latency = testutil.MOCK_LATENCY_THRESHOLD * 1.5
	mm.RelDeploymentUtil = 2
	mm.NodeUsages = map[string]int64{
		"node1": 900,
		"node2": 500,
	}

	// test
	a := UnitMakeAutoscaler(0.2, 0.85, testutil.MOCK_DEPLOYMENT_NAMESPACE, 500, 100, mm)
	err := a.Init()
	testutil.AssertNoError(err, t)

	err = a.RunRound()
	testutil.AssertNoError(err, t)

	testutil.AssertAction(mm, t, testutil.Action{Type: testutil.ChangeReplicaCountAction, Namespace: testutil.MOCK_DEPLOYMENT_NAMESPACE, DeploymentName: testutil.MOCK_DEPLOYMENT_NAME, ReplicaCt: 4})
	testutil.AssertAction(mm, t, testutil.Action{Type: testutil.VscaleAction, PodName: "pod1", ContainerName: "container", CpuRequests: "450m"})
	testutil.AssertAction(mm, t, testutil.Action{Type: testutil.VscaleAction, PodName: "pod2", ContainerName: "container", CpuRequests: "450m"})
	testutil.AssertAction(mm, t, testutil.Action{Type: testutil.VscaleAction, PodName: "pod3", ContainerName: "container", CpuRequests: "450m"})
	testutil.AssertAction(mm, t, testutil.Action{Type: testutil.VscaleAction, PodName: "pod4", ContainerName: "container", CpuRequests: "450m"})
	testutil.AssertNoActions(mm, t)

	testutil.AssertPodListsEqual(mm.Pods, correctEndPods, t)
}

func TestUnit_BasicVscaleDown(t *testing.T) {
	// values to test
	correctEndPods := map[string]testutil.PodData{
		"pod1": {PodName: "pod1", NodeName: "node1", ContainerName: "container", CpuRequests: 283}, // ceil(240 / 0.85) = 283
		"pod2": {PodName: "pod2", NodeName: "node1", ContainerName: "container", CpuRequests: 283},
		"pod3": {PodName: "pod3", NodeName: "node2", ContainerName: "container", CpuRequests: 283},
	}

	// setup
	mm := testutil.CreateSimpleMockMetrics() // start 3 pods at 300 each
	mm.Latency = testutil.MOCK_LATENCY_THRESHOLD * 0.9
	mm.RelDeploymentUtil = 0.8 // default alloc is 900

	// test
	a := UnitMakeAutoscaler(0.2, 0.85, testutil.MOCK_DEPLOYMENT_NAMESPACE, 300, 100, mm)
	err := a.Init()
	testutil.AssertNoError(err, t)

	err = a.RunRound()
	testutil.AssertNoError(err, t)

	testutil.AssertAction(mm, t, testutil.Action{Type: testutil.VscaleAction, PodName: "pod1", ContainerName: "container", CpuRequests: "283m"})
	testutil.AssertAction(mm, t, testutil.Action{Type: testutil.VscaleAction, PodName: "pod2", ContainerName: "container", CpuRequests: "283m"})
	testutil.AssertAction(mm, t, testutil.Action{Type: testutil.VscaleAction, PodName: "pod3", ContainerName: "container", CpuRequests: "283m"})
	testutil.AssertNoActions(mm, t)

	testutil.AssertPodListsEqual(mm.Pods, correctEndPods, t)
}

func TestUnit_BasicHscaleDown(t *testing.T) {
	// values to test
	correctEndPods := map[string]testutil.PodData{
		"pod1": {PodName: "pod1", NodeName: "node1", ContainerName: "container", CpuRequests: 265}, // ceil(225 / 0.85) = 265
		"pod2": {PodName: "pod2", NodeName: "node1", ContainerName: "container", CpuRequests: 265},
	}

	// setup
	mm := testutil.CreateSimpleMockMetrics() // start 3 pods at 300 each
	mm.Latency = testutil.MOCK_LATENCY_THRESHOLD * 0.9
	mm.RelDeploymentUtil = 0.5 // default alloc is 900

	// test
	a := UnitMakeAutoscaler(0.2, 0.85, testutil.MOCK_DEPLOYMENT_NAMESPACE, 300, 100, mm)
	err := a.Init()
	testutil.AssertNoError(err, t)

	err = a.RunRound()
	testutil.AssertNoError(err, t)

	testutil.AssertAction(mm, t, testutil.Action{Type: testutil.ChangeReplicaCountAction, Namespace: testutil.MOCK_DEPLOYMENT_NAMESPACE, DeploymentName: testutil.MOCK_DEPLOYMENT_NAME, ReplicaCt: 2})
	testutil.AssertAction(mm, t, testutil.Action{Type: testutil.VscaleAction, PodName: "pod1", ContainerName: "container", CpuRequests: "265m"})
	testutil.AssertAction(mm, t, testutil.Action{Type: testutil.VscaleAction, PodName: "pod2", ContainerName: "container", CpuRequests: "265m"})
	testutil.AssertNoActions(mm, t)

	testutil.AssertPodListsEqual(mm.Pods, correctEndPods, t)
}

func TestUnit_NoCongestion(t *testing.T) {
	// values to test
	correctEndPods := map[string]testutil.PodData{
		"pod1": {PodName: "pod1", NodeName: "node1", ContainerName: "container", CpuRequests: 300},
		"pod2": {PodName: "pod2", NodeName: "node1", ContainerName: "container", CpuRequests: 300},
		"pod3": {PodName: "pod3", NodeName: "node2", ContainerName: "container", CpuRequests: 300},
	}

	// setup
	mm := testutil.CreateSimpleMockMetrics() // start 3 pods at 300 each
	mm.Latency = testutil.MOCK_LATENCY_THRESHOLD * 1.5
	mm.RelDeploymentUtil = 0.9
	mm.NodeUsages = map[string]int64{
		"node1": 600,
		"node2": 300,
	}

	// test
	a := UnitMakeAutoscaler(0.2, 0.85, testutil.MOCK_DEPLOYMENT_NAMESPACE, 400, 100, mm)
	err := a.Init()
	testutil.AssertNoError(err, t)

	err = a.RunRound()
	testutil.AssertNoError(err, t)

	testutil.AssertNoActions(mm, t)

	testutil.AssertPodListsEqual(mm.Pods, correctEndPods, t)
}

func TestUnit_PodMove(t *testing.T) {
	// values to test
	correctEndPods := map[string]testutil.PodData{
		"pod2": {PodName: "pod2", NodeName: "node1", ContainerName: "container", CpuRequests: 330},
		"pod3": {PodName: "pod3", NodeName: "node2", ContainerName: "container", CpuRequests: 330},
		"pod4": {PodName: "pod4", NodeName: "node2", ContainerName: "container", CpuRequests: 330},
	}

	// setup
	mm := testutil.CreateSimpleMockMetrics() // start 3 pods at 300 each
	mm.Latency = testutil.MOCK_LATENCY_THRESHOLD * 1.5
	mm.RelDeploymentUtil = 1.1
	mm.NodeUsages = map[string]int64{
		"node1": 1000,
		"node2": 500,
	}
	mm.NodeAllocables = map[string]int64{
		"node1": 10,
		"node2": 700,
	}

	// test
	a := UnitMakeAutoscaler(0.2, 0.85, testutil.MOCK_DEPLOYMENT_NAMESPACE, 400, 100, mm)
	err := a.Init()
	testutil.AssertNoError(err, t)

	err = a.RunRound()
	testutil.AssertNoError(err, t)

	testutil.AssertAction(mm, t, testutil.Action{Type: testutil.ChangeReplicaCountAction, Namespace: testutil.MOCK_DEPLOYMENT_NAMESPACE, DeploymentName: testutil.MOCK_DEPLOYMENT_NAME, ReplicaCt: 4})
	testutil.AssertAction(mm, t, testutil.Action{Type: testutil.DeletePodAction, Namespace: testutil.MOCK_DEPLOYMENT_NAMESPACE, PodName: "pod1"})
	testutil.AssertAction(mm, t, testutil.Action{Type: testutil.ChangeReplicaCountAction, Namespace: testutil.MOCK_DEPLOYMENT_NAMESPACE, DeploymentName: testutil.MOCK_DEPLOYMENT_NAME, ReplicaCt: 3})
	testutil.AssertAction(mm, t, testutil.Action{Type: testutil.VscaleAction, PodName: "pod2", ContainerName: "container", CpuRequests: "330m"})
	testutil.AssertAction(mm, t, testutil.Action{Type: testutil.VscaleAction, PodName: "pod3", ContainerName: "container", CpuRequests: "330m"})
	testutil.AssertAction(mm, t, testutil.Action{Type: testutil.VscaleAction, PodName: "pod4", ContainerName: "container", CpuRequests: "330m"})
	testutil.AssertNoActions(mm, t)

	testutil.AssertPodListsEqual(mm.Pods, correctEndPods, t)
}

// error handling?
