package main

import (
	"testing"

	util "github.com/tholiang/podoscaler/scalers/util"

	kube_client "k8s.io/client-go/kubernetes"
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
	mm.MockVScale = func(m *util.MockMetrics, clientset kube_client.Interface, podname string, containername string, cpurequests string) error {
		t.Errorf("should not be vscaling")
		return nil
	}
	mm.MockChangeReplicaCount = func(m *util.MockMetrics, namespace string, deploymentName string, replicaCt int, clientset kube_client.Interface) error {
		t.Errorf("should not be hscaling")
		return nil
	}

	// test
	a := MakeAutoscaler(0.2, 0.85, "namespace", 500, 100, mm)
	err := a.Init()
	util.AssertNoError(err, t)

	err = a.RunRound()
	util.AssertNoError(err, t)
}

func TestBasicVscaleUp(t *testing.T) {
	// values to test
	vscaleCounters := map[string]int{}
	correctVscaleCounters := map[string]int{
		"pod1": 1,
		"pod2": 1,
		"pod3": 1,
	}
	correctEndPods := map[string]util.PodData{
		"pod1": {PodName: "pod1", NodeName: "node1", ContainerName: "container", CpuRequests: 330},
		"pod2": {PodName: "pod2", NodeName: "node1", ContainerName: "container", CpuRequests: 330},
		"pod3": {PodName: "pod3", NodeName: "node2", ContainerName: "container", CpuRequests: 330},
	}

	// setup
	mm := util.CreateSimpleMockMetrics()
	mm.Latency = util.MOCK_LATENCY_THRESHOLD * 1.5
	mm.DeploymentUtil = int64(float64(util.GetDeploymentAlloc(mm.Pods)) * 1.1)
	mm.NodeUsages = map[string]int64{
		"node1": 900,
		"node2": 500,
	}
	mm.MockVScale = func(m *util.MockMetrics, clientset kube_client.Interface, podname string, containername string, cpurequests string) error {
		_, ok := vscaleCounters[podname]
		if !ok {
			vscaleCounters[podname] = 1
		} else {
			vscaleCounters[podname]++
		}
		return util.MockVScale(m, clientset, podname, containername, cpurequests)
	}
	mm.MockChangeReplicaCount = func(m *util.MockMetrics, namespace string, deploymentName string, replicaCt int, clientset kube_client.Interface) error {
		t.Errorf("should not be hscaling")
		return nil
	}

	// test
	a := MakeAutoscaler(0.2, 0.85, "namespace", 400, 100, mm)
	err := a.Init()
	util.AssertNoError(err, t)

	err = a.RunRound()
	util.AssertNoError(err, t)

	util.AssertStringIntMapsEqual(vscaleCounters, correctVscaleCounters, t)
	util.AssertPodListsEqual(mm.Pods, correctEndPods, t)
}

func TestBasicHscaleUp(t *testing.T) {
}

// basic vscale down

// basic hscale down

// no congestion

// pod move

// error handling?
