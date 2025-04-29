package main

import (
	"testing"

	util "github.com/tholiang/podoscaler/scalers/util"
)

func MakeAutoscaler(node_avail_threshold float64, downscale_threshold float64, namespace string, maps int64, latency_threshold int64, metrics AutoscalerMetrics) Autoscaler {
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
func TestBasicStable(t *testing.T) {
	// setup
	mm := util.CreateSimpleMockMetrics()

	// test
	a := MakeAutoscaler(0.2, 0.85, util.MOCK_DEPLOYMENT_NAMESPACE, 500, 100, mm)
	err := a.Init()
	util.AssertNoError(err, t)

	err = a.RunRound()
	util.AssertNoError(err, t)

	util.AssertNoActions(mm, t)
}

func TestBasicVscaleUp(t *testing.T) {
	// values to test
	correctEndPods := map[string]util.PodData{
		"pod1": {PodName: "pod1", NodeName: "node1", ContainerName: "container", CpuRequests: 330},
		"pod2": {PodName: "pod2", NodeName: "node1", ContainerName: "container", CpuRequests: 330},
		"pod3": {PodName: "pod3", NodeName: "node2", ContainerName: "container", CpuRequests: 330},
	}

	// setup
	mm := util.CreateSimpleMockMetrics() // start 3 pods at 300 each
	mm.Latency = util.MOCK_LATENCY_THRESHOLD * 1.5
	mm.DeploymentUtil = 990
	mm.NodeUsages = map[string]int64{
		"node1": 900,
		"node2": 500,
	}

	// test
	a := MakeAutoscaler(0.2, 0.85, util.MOCK_DEPLOYMENT_NAMESPACE, 400, 100, mm)
	err := a.Init()
	util.AssertNoError(err, t)

	err = a.RunRound()
	util.AssertNoError(err, t)

	util.AssertAction(mm, t, util.Action{Type: util.VscaleAction, PodName: "pod1", ContainerName: "container", CpuRequests: "330m"})
	util.AssertAction(mm, t, util.Action{Type: util.VscaleAction, PodName: "pod2", ContainerName: "container", CpuRequests: "330m"})
	util.AssertAction(mm, t, util.Action{Type: util.VscaleAction, PodName: "pod3", ContainerName: "container", CpuRequests: "330m"})
	util.AssertNoActions(mm, t)

	util.AssertPodListsEqual(mm.Pods, correctEndPods, t)
}

func TestBasicHscaleUp(t *testing.T) {
	// values to test
	correctEndPods := map[string]util.PodData{
		"pod1": {PodName: "pod1", NodeName: "node1", ContainerName: "container", CpuRequests: 450},
		"pod2": {PodName: "pod2", NodeName: "node1", ContainerName: "container", CpuRequests: 450},
		"pod3": {PodName: "pod3", NodeName: "node2", ContainerName: "container", CpuRequests: 450},
		"pod4": {PodName: "pod4", NodeName: "node2", ContainerName: "container", CpuRequests: 450},
	}

	// setup
	mm := util.CreateSimpleMockMetrics() // start 3 pods at 300 each
	mm.Latency = util.MOCK_LATENCY_THRESHOLD * 1.5
	mm.DeploymentUtil = 1800
	mm.NodeUsages = map[string]int64{
		"node1": 900,
		"node2": 500,
	}

	// test
	a := MakeAutoscaler(0.2, 0.85, util.MOCK_DEPLOYMENT_NAMESPACE, 500, 100, mm)
	err := a.Init()
	util.AssertNoError(err, t)

	err = a.RunRound()
	util.AssertNoError(err, t)

	util.AssertAction(mm, t, util.Action{Type: util.ChangeReplicaCountAction, Namespace: util.MOCK_DEPLOYMENT_NAMESPACE, DeploymentName: util.MOCK_DEPLOYMENT_NAME, ReplicaCt: 4})
	util.AssertAction(mm, t, util.Action{Type: util.VscaleAction, PodName: "pod1", ContainerName: "container", CpuRequests: "450m"})
	util.AssertAction(mm, t, util.Action{Type: util.VscaleAction, PodName: "pod2", ContainerName: "container", CpuRequests: "450m"})
	util.AssertAction(mm, t, util.Action{Type: util.VscaleAction, PodName: "pod3", ContainerName: "container", CpuRequests: "450m"})
	util.AssertAction(mm, t, util.Action{Type: util.VscaleAction, PodName: "pod4", ContainerName: "container", CpuRequests: "450m"})
	util.AssertNoActions(mm, t)

	util.AssertPodListsEqual(mm.Pods, correctEndPods, t)
}

func TestBasicVscaleDown(t *testing.T) {
	// values to test
	correctEndPods := map[string]util.PodData{
		"pod1": {PodName: "pod1", NodeName: "node1", ContainerName: "container", CpuRequests: 295}, // ceil(250 / 0.85) = 295
		"pod2": {PodName: "pod2", NodeName: "node1", ContainerName: "container", CpuRequests: 295},
		"pod3": {PodName: "pod3", NodeName: "node2", ContainerName: "container", CpuRequests: 295},
	}

	// setup
	mm := util.CreateSimpleMockMetrics() // start 3 pods at 300 each
	mm.Latency = util.MOCK_LATENCY_THRESHOLD * 0.9
	mm.DeploymentUtil = int64(750) // default alloc is 900

	// test
	a := MakeAutoscaler(0.2, 0.85, util.MOCK_DEPLOYMENT_NAMESPACE, 300, 100, mm)
	err := a.Init()
	util.AssertNoError(err, t)

	err = a.RunRound()
	util.AssertNoError(err, t)

	util.AssertAction(mm, t, util.Action{Type: util.VscaleAction, PodName: "pod1", ContainerName: "container", CpuRequests: "295m"})
	util.AssertAction(mm, t, util.Action{Type: util.VscaleAction, PodName: "pod2", ContainerName: "container", CpuRequests: "295m"})
	util.AssertAction(mm, t, util.Action{Type: util.VscaleAction, PodName: "pod3", ContainerName: "container", CpuRequests: "295m"})
	util.AssertNoActions(mm, t)

	util.AssertPodListsEqual(mm.Pods, correctEndPods, t)
}

func TestBasicHscaleDown(t *testing.T) {
	// values to test
	correctEndPods := map[string]util.PodData{
		"pod1": {PodName: "pod1", NodeName: "node1", ContainerName: "container", CpuRequests: 295}, // ceil(250 / 0.85) = 295
		"pod2": {PodName: "pod2", NodeName: "node1", ContainerName: "container", CpuRequests: 295},
	}

	// setup
	mm := util.CreateSimpleMockMetrics() // start 3 pods at 300 each
	mm.Latency = util.MOCK_LATENCY_THRESHOLD * 0.9
	mm.DeploymentUtil = int64(500) // default alloc is 900

	// test
	a := MakeAutoscaler(0.2, 0.85, util.MOCK_DEPLOYMENT_NAMESPACE, 300, 100, mm)
	err := a.Init()
	util.AssertNoError(err, t)

	err = a.RunRound()
	util.AssertNoError(err, t)

	util.AssertAction(mm, t, util.Action{Type: util.ChangeReplicaCountAction, Namespace: util.MOCK_DEPLOYMENT_NAMESPACE, DeploymentName: util.MOCK_DEPLOYMENT_NAME, ReplicaCt: 2})
	util.AssertAction(mm, t, util.Action{Type: util.VscaleAction, PodName: "pod1", ContainerName: "container", CpuRequests: "295m"})
	util.AssertAction(mm, t, util.Action{Type: util.VscaleAction, PodName: "pod2", ContainerName: "container", CpuRequests: "295m"})
	util.AssertNoActions(mm, t)

	util.AssertPodListsEqual(mm.Pods, correctEndPods, t)
}

// no congestion

// pod move

// error handling?
