package main

import (
	"slices"
	"testing"

	util "github.com/tholiang/podoscaler/scalers/util"

	kube_client "k8s.io/client-go/kubernetes"
)

/* FULL MOCK UNIT TESTS - DOESN'T CREATE ANY PODS (DOESN'T EVEN NEED TO BE RUN IN K8S) */
func TestBasicStable(t *testing.T) {
	// setup
	mm := util.CreateSimpleMockMetrics()
	mockVScale := func(clientset kube_client.Interface, podname string, containername string, cpurequests string) error {
		t.Errorf("should not be vscaling")
		return nil
	}
	mockChangeReplicaCount := func(namespace string, deploymentName string, replicaCt int, clientset kube_client.Interface) error {
		t.Errorf("should not be hscaling")
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
}

func TestBasicVscaleUp(t *testing.T) {
	// setup
	vscaleCounters := map[string]int{}
	correctVscaleCounters := map[string]int{
		"pod1": 1,
		"pod2": 1,
		"pod3": 1,
	}

	mm := util.CreateSimpleMockMetrics()
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
		t.Errorf("should not be hscaling")
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
}
