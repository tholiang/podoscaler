package autoscalertest

import (
	"fmt"
	"time"

	"github.com/tholiang/podoscaler/scalers/autoscaler"
	"github.com/tholiang/podoscaler/scalers/util"
)

func integration_make_autoscaler(node_avail_threshold float64, downscale_threshold float64, namespace string, Maps int64, LatencyThreshold int64, mm *MockMetrics) (autoscaler.Autoscaler, error) {
	a := autoscaler.Autoscaler{
		PrometheusUrl:                 autoscaler.DEFAULT_PROMETHEUS_URL,
		MinNodeAvailabilityThreshold:  node_avail_threshold,
		DownscaleUtilizationThreshold: downscale_threshold,
		DeploymentNamespace:           namespace,
		Maps:                          Maps,
		LatencyThreshold:              LatencyThreshold,
		Metrics:                       mm,
	}
	err := a.Init()
	if err != nil {
		return a, err
	}

	_, node_cap, err := util.GetNodeAllocableAndCapacity(a.Clientset, "minikube")
	IntAssertNoError(err)
	mm.NodeCapacities = map[string]int64{
		"minikube": node_cap, // workaround
	}

	return a, nil
}

func reset_dummy(a autoscaler.Autoscaler) {
	fmt.Println("<<< reseting dummy deployment >>>")
	err := util.ChangeReplicaCount("default", "dummy", 1, a.Clientset)
	IntAssertNoError(err)

	time.Sleep(1 * time.Second)

	podlist, err := util.GetPodListForDeployment(a.Clientset, "dummy", "default")
	IntAssertNoError(err)
	IntAssertIntsEqual(len(podlist.Items), 1)

	podname := podlist.Items[0].Name
	err = util.VScale(a.Clientset, podname, "dummy-container", "300m")
	IntAssertNoError(err)

	time.Sleep(1 * time.Second)

	podlist, err = util.GetPodListForDeployment(a.Clientset, "dummy", "default")
	IntAssertNoError(err)
	IntAssertIntsEqual(len(podlist.Items), 1)
	newsize := podlist.Items[0].Spec.Containers[0].Resources.Requests.Cpu().MilliValue()
	IntAssertIntsEqual(int(newsize), 300)

	fmt.Println("<<< reset successful >>>")
	fmt.Println()
}

/* INTEGRATION TESTS TO BE RUN IN A CLUSTER */
/* PATCH UTILIZATION AND LATENCY */

/* all start with 1 pod at 300m cpu */

func IntegrationTest_BasicStable() {
	// setup
	mm := CreateIntMockMetrics()

	// test
	a, err := integration_make_autoscaler(0.2, 0.85, "default", 500, 100, mm)
	IntAssertNoError(err)

	reset_dummy(a)

	err = a.RunRound()
	IntAssertNoError(err)

	IntAssertNoActions(mm)

	fmt.Println("<<< Test_BasicStable passed! >>>")
	fmt.Println()
}

func IntegrationTest_BasicVscaleUp() {
	// values to test
	correctEndPods := []PodData{
		{PodName: "pod", NodeName: "minikube", ContainerName: "container", CpuRequests: 330}, // string values don't matter here
	}

	// setup
	mm := CreateIntMockMetrics()
	mm.Latency = MOCK_LATENCY_THRESHOLD * 1.5
	mm.RelDeploymentUtil = 1.1
	mm.RelNodeUsages = map[string]float64{
		"minikube": 0.9,
	}

	// test
	a, err := integration_make_autoscaler(0.2, 0.85, "default", 400, 100, mm)
	IntAssertNoError(err)

	reset_dummy(a)

	err = a.RunRound()
	IntAssertNoError(err)

	IntAssertAction(mm, Action{Type: VscaleAction, PodName: "pod", ContainerName: "container", CpuRequests: "330m"})
	IntAssertNoActions(mm)

	podlist, err := util.GetPodListForDeployment(a.Clientset, mm.DeploymentName, mm.DeploymentNamespace)
	IntAssertNoError(err)
	IntAssertPodListsEqual(podlist, correctEndPods)

	fmt.Println("<<< Test_BasicVscaleUp passed! >>>")
	fmt.Println()
}

func IntegrationTest_BasicHscaleUp() {
	// values to test
	correctEndPods := []PodData{
		{PodName: "pod1", NodeName: "minikube", ContainerName: "container", CpuRequests: 450},
		{PodName: "pod2", NodeName: "minikube", ContainerName: "container", CpuRequests: 450},
	}

	// setup
	mm := CreateIntMockMetrics()
	mm.Latency = MOCK_LATENCY_THRESHOLD * 1.5
	mm.RelDeploymentUtil = 3
	mm.RelNodeUsages = map[string]float64{
		"minikube": 0.9,
	}

	// test
	a, err := integration_make_autoscaler(0.2, 0.85, MOCK_DEPLOYMENT_NAMESPACE, 500, 100, mm)
	IntAssertNoError(err)

	reset_dummy(a)

	err = a.RunRound()
	IntAssertNoError(err)

	IntAssertAction(mm, Action{Type: ChangeReplicaCountAction, Namespace: "default", DeploymentName: "dummy", ReplicaCt: 2})
	IntAssertAction(mm, Action{Type: VscaleAction, PodName: "pod1", ContainerName: "container", CpuRequests: "450m"})
	IntAssertAction(mm, Action{Type: VscaleAction, PodName: "pod2", ContainerName: "container", CpuRequests: "450m"})
	IntAssertNoActions(mm)

	podlist, err := util.GetPodListForDeployment(a.Clientset, mm.DeploymentName, mm.DeploymentNamespace)
	IntAssertNoError(err)
	IntAssertPodListsEqual(podlist, correctEndPods)

	fmt.Println("<<< Test_BasicHscaleUp passed! >>>")
	fmt.Println()
}

func RunIntegrationTests() {
	IntegrationTest_BasicStable()
	IntegrationTest_BasicVscaleUp()
	IntegrationTest_BasicHscaleUp()

	fmt.Println("<<< TESTS PASSED SUCCESSFULLY >>>")
}
