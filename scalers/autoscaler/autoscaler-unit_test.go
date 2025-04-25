package main

import (
	"fmt"
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
	vscaleCounter := 0
	correctVscaleCounter := 1

	mm := util.CreateSimpleMockMetrics()
	mm.MockGetLatencyMetrics = util.SimpleOverLatencyMetrics
	mm.MockGetDeploymentUtilAndAlloc = util.SimpleOverDeploymentUtilAndAlloc
	mm.MockGetNodeAllocableAndCapacity = util.SimpleCongestedNodeAllocableAndCapacity
	mockVScale := func(clientset kube_client.Interface, podname string, containername string, cpurequests string) error {
		fmt.Println("here")
		vscaleCounter++
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

	if vscaleCounter != correctVscaleCounter {
		t.Errorf("vscaled %d times, expected %d", vscaleCounter, correctVscaleCounter)
	}
}
