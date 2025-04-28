package main

import (
	"fmt"
	"slices"
	"testing"

	util "github.com/tholiang/podoscaler/scalers/util"

	v1 "k8s.io/api/core/v1"
	kube_client "k8s.io/client-go/kubernetes"
)

/* FULL MOCK UNIT TESTS - DOESN'T CREATE ANY PODS (DOESN'T EVEN NEED TO BE RUN IN K8S) */
func TestBasicStable(t *testing.T) {
	// setup
	mm := util.CreateSimpleMockMetrics()
	patchedVScale := func(m *util.MockMetrics, clientset kube_client.Interface, podname string, containername string, cpurequests string) error {
		t.Errorf("should not be vscaling")
		return nil
	}
	patchedChangeReplicaCount := func(m *util.MockMetrics, namespace string, deploymentName string, replicaCt int, clientset kube_client.Interface) error {
		t.Errorf("should not be hscaling")
		return nil
	}

	mm.MockVScale = patchedVScale
	mm.MockChangeReplicaCount = patchedChangeReplicaCount

	// test
	a := Autoscaler{}
	err := a.Init(mm)
	if err != nil {
		t.Errorf("%s", err.Error())
	}

	err = a.RunRound()
	if err != nil {
		t.Errorf("%s", err.Error())
	}
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
	mm.Latency = util.MOCK_LATENCY_THRESHOLD * 1.1
	mm.DeploymentUtil = int64(float64(util.GetDeploymentAlloc(mm.Pods)) * 1.1)
	mm.NodeAllocables = map[string]int64{
		"node1": 100,
		"node2": 400,
	}
	mm.NodeCapacities = map[string]int64{
		"node1": 700,
		"node2": 700,
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
	a := Autoscaler{}
	err := a.Init(mm)
	util.AssertNoError(err, t)

	err = a.RunRound()
	util.AssertNoError(err, t)

	util.AssertStringIntMapsEqual(vscaleCounters, correctVscaleCounters, t)
	util.AssertPodListsEqual(mm.Pods, correctEndPods, t)
}

func TestBasicHscaleUp(t *testing.T) {
	// values to test
	hscaleCounter := 0
	correctHscales := 1
	numReplicas := 3
	correctNumReplicas := 4
	vscaleCounters := map[string]int{}
	correctVscaleCounters := map[string]int{
		"pod1": 1,
		"pod2": 1,
		"pod3": 1,
		"pod4": 1,
	}

	// setup
	podlist := new(v1.PodList)
	podlist.Items = []v1.Pod{
		util.MakePod("pod1", "node1", 300),
		util.MakePod("pod2", "node1", 300),
		util.MakePod("pod3", "node2", 300),
	}
	mockGetPodlist := func(clientset kube_client.Interface, deploymentName, namespace string) (*v1.PodList, error) {
		return podlist, nil
	}

	mm := util.CreateSimpleMockMetrics()
	mm.MockGetPodListForDeployment = mockGetPodlist
	mm.MockGetLatencyMetrics = util.SimpleOverLatencyMetrics
	mm.MockGetDeploymentUtilAndAlloc = util.SimpleOverDeploymentUtilAndAlloc
	mm.MockGetNodeAllocableAndCapacity = util.SimpleCongestedNodeAllocableAndCapacity
	mockVScale := func(clientset kube_client.Interface, podname string, containername string, cpurequests string) error {
		_, ok := vscaleCounters[podname]
		if !ok {
			vscaleCounters[podname] = 1
		} else {
			vscaleCounters[podname]++
		}
		return nil
	}
	mockChangeReplicaCount := func(namespace string, deploymentName string, replicaCt int, clientset kube_client.Interface) error {
		hscaleCounter++
		for i := numReplicas + 1; i <= replicaCt; i++ {
			podlist.Items = append(podlist.Items, util.MakePod(fmt.Sprintf("pod%d", i), "node2", 300))
		}
		numReplicas = replicaCt
		return nil
	}

	mm.MockVScale = mockVScale
	mm.MockChangeReplicaCount = mockChangeReplicaCount

	// test
	a := Autoscaler{}
	err := a.Init(mm)
	if err != nil {
		t.Errorf("%s", err.Error())
	}

	err = a.RunRound()
	if err != nil {
		t.Errorf("%s", err.Error())
	}

	testkeys := util.GetStringIntMapKeys(vscaleCounters)
	correctkeys := util.GetStringIntMapKeys(correctVscaleCounters)
	if !slices.Equal(testkeys, correctkeys) {
		t.Errorf("incorrect pods were scaled")
	}

	for _, k := range testkeys {
		if vscaleCounters[k] != correctVscaleCounters[k] {
			t.Errorf("incorrect number of vscales for pod %s, expected %d vscales, got %d", k, vscaleCounters[k], correctVscaleCounters[k])
		}
	}

	if hscaleCounter != correctHscales {
		t.Errorf("incorrect number of hscales, expected %d, got %d", correctHscales, hscaleCounter)
	}

	if numReplicas != correctNumReplicas {
		t.Errorf("incorrect number of replicas at finish, expected %d, got %d", correctNumReplicas, numReplicas)
	}
}

// basic vscale down

// basic hscale down

// no congestion

// pod move

// error handling?
