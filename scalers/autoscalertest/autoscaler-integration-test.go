package autoscalertest

import (
	"fmt"
	"time"

	"github.com/tholiang/podoscaler/scalers/autoscaler"
	"github.com/tholiang/podoscaler/scalers/util"
	"k8s.io/client-go/kubernetes"
)

func integration_make_autoscaler(node_avail_threshold float64, downscale_threshold float64, namespace string, Maps int64, LatencyThreshold int64, mm *MockMetrics) (autoscaler.Autoscaler, error) {
	a := autoscaler.Autoscaler{
		PrometheusUrl:                 autoscaler.DEFAULT_PROMETHEUS_URL,
		MinNodeAvailabilityThreshold:  node_avail_threshold,
		DownscaleUtilizationThreshold: downscale_threshold,
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

func reset_dummy(clientset kubernetes.Interface) {
	fmt.Println("<<< reseting dummy deployment >>>")
	err := util.ChangeReplicaCount("default", "dummy", 1, clientset)
	IntAssertNoError(err)

	podlist, err := util.GetReadyPodListForDeployment(clientset, "dummy", "default")
	IntAssertNoError(err)
	IntAssertIntsEqual(1, len(podlist))

	podname := podlist[0].Name
	err = util.VScale(clientset, podname, "dummy-container", "300m")
	IntAssertNoError(err)

	time.Sleep(100 * time.Millisecond)

	podlist, err = util.GetReadyPodListForDeployment(clientset, "dummy", "default")
	IntAssertNoError(err)
	IntAssertIntsEqual(1, len(podlist))
	newsize := podlist[0].Spec.Containers[0].Resources.Requests.Cpu().MilliValue()
	IntAssertIntsEqual(300, int(newsize))

	fmt.Println("<<< reset successful >>>")
	fmt.Println()
}

func add_dummy_pod(clientset kubernetes.Interface) {
	fmt.Println("<<< \"manually\" adding new dummy pod >>>")
	err := util.ChangeReplicaCount("default", "dummy", 1, clientset)
	IntAssertNoError(err)

	podlist, err := util.GetReadyPodListForDeployment(clientset, "dummy", "default")
	IntAssertNoError(err)
	starting_pod_ct := len(podlist)

	err = util.ChangeReplicaCount("default", "dummy", starting_pod_ct+1, clientset)
	IntAssertNoError(err)

	// wait for hscale to work
	timeout_millis := 5000
	for range timeout_millis / 500 { // gross
		podlist, err := util.GetReadyPodListForDeployment(clientset, "dummy", "default")
		IntAssertNoError(err)
		if len(podlist) == starting_pod_ct+1 {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	podlist, err = util.GetReadyPodListForDeployment(clientset, "dummy", "default")
	IntAssertNoError(err)
	IntAssertIntsEqual(starting_pod_ct+1, len(podlist))

	// check sizes
	for i := range starting_pod_ct + 1 {
		podsize := podlist[i].Spec.Containers[0].Resources.Requests.Cpu().MilliValue()
		IntAssertIntsEqual(300, int(podsize))
	}

	fmt.Println("<<< addition successful >>>")
	fmt.Println()
}

/* INTEGRATION TESTS TO BE RUN IN A CLUSTER */
/* PATCH UTILIZATION AND LATENCY */

/* all start with 1 pod at 300m cpu */

func IntegrationTest_BasicStable() {
	name := "Test_BasicStable"
	fmt.Printf("<<< %s >>>\n", name)

	// setup
	mm := CreateIntMockMetrics()

	// test
	a, err := integration_make_autoscaler(0.2, 0.85, "default", 500, 100, mm)
	IntAssertNoError(err)

	reset_dummy(a.Clientset)

	err = a.RunRound()
	IntAssertNoError(err)

	IntSummarizeActions(mm)
	IntAssertNoActions(mm)

	fmt.Printf("<<< %s passed! >>>\n\n", name)
}

func IntegrationTest_BasicVscaleUp() {
	name := "Test_BasicVscaleUp"
	fmt.Printf("<<< %s >>>\n", name)

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

	reset_dummy(a.Clientset)

	err = a.RunRound()
	IntAssertNoError(err)

	IntSummarizeActions(mm)

	IntAssertAction(mm, Action{Type: VscaleAction, PodName: "pod", ContainerName: "container", CpuRequests: "330m"})
	IntAssertNoActions(mm)

	podlist, err := util.GetReadyPodListForDeployment(a.Clientset, mm.DeploymentName, mm.DeploymentNamespace)
	IntAssertNoError(err)
	IntAssertPodListsEqual(podlist, correctEndPods)

	fmt.Printf("<<< %s passed! >>>\n\n", name)
}

func IntegrationTest_BasicHscaleUp() {
	name := "Test_BasicHscaleUp"
	fmt.Printf("<<< %s >>>\n", name)

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

	reset_dummy(a.Clientset)

	err = a.RunRound()
	IntAssertNoError(err)

	IntSummarizeActions(mm)

	IntAssertAction(mm, Action{Type: ChangeReplicaCountAction, Namespace: "default", DeploymentName: "dummy", ReplicaCt: 2})
	IntAssertAction(mm, Action{Type: VscaleAction, PodName: "pod1", ContainerName: "container", CpuRequests: "450m"})
	IntAssertAction(mm, Action{Type: VscaleAction, PodName: "pod2", ContainerName: "container", CpuRequests: "450m"})
	IntAssertNoActions(mm)

	podlist, err := util.GetReadyPodListForDeployment(a.Clientset, mm.DeploymentName, mm.DeploymentNamespace)
	IntAssertNoError(err)
	IntAssertPodListsEqual(podlist, correctEndPods)

	fmt.Printf("<<< %s passed! >>>\n\n", name)
}

func IntegrationTest_BasicVscaleDown() {
	name := "Test_BasicVscaleDown"
	fmt.Printf("<<< %s >>>\n", name)

	// values to test
	correctEndPods := []PodData{
		{PodName: "pod1", NodeName: "minikube", ContainerName: "container", CpuRequests: 177}, // ceil(150 / 0.85) = 177
	}

	// setup
	mm := CreateIntMockMetrics()
	mm.Latency = MOCK_LATENCY_THRESHOLD * 0.5
	mm.RelDeploymentUtil = 0.5

	// test
	a, err := integration_make_autoscaler(0.2, 0.85, MOCK_DEPLOYMENT_NAMESPACE, 500, 100, mm)
	IntAssertNoError(err)

	reset_dummy(a.Clientset)

	err = a.RunRound()
	IntAssertNoError(err)

	IntSummarizeActions(mm)

	IntAssertAction(mm, Action{Type: VscaleAction, PodName: "pod", ContainerName: "container", CpuRequests: "177m"})
	IntAssertNoActions(mm)

	podlist, err := util.GetReadyPodListForDeployment(a.Clientset, mm.DeploymentName, mm.DeploymentNamespace)
	IntAssertNoError(err)
	IntAssertPodListsEqual(podlist, correctEndPods)

	fmt.Printf("<<< %s passed! >>>\n\n", name)
}

func IntegrationTest_BasicHscaleDown() {
	name := "Test_BasicHscaleDown"
	fmt.Printf("<<< %s >>>\n", name)

	// values to test
	correctEndPods := []PodData{
		{PodName: "pod1", NodeName: "minikube", ContainerName: "container", CpuRequests: 353}, // ceil(300 / 0.85) = 353
	}

	// setup
	mm := CreateIntMockMetrics()
	mm.Latency = MOCK_LATENCY_THRESHOLD * 0.5
	mm.RelDeploymentUtil = 0.5

	// test
	a, err := integration_make_autoscaler(0.2, 0.85, MOCK_DEPLOYMENT_NAMESPACE, 500, 100, mm)
	IntAssertNoError(err)

	reset_dummy(a.Clientset)
	add_dummy_pod(a.Clientset) // start two pods at 300m

	err = a.RunRound()
	IntAssertNoError(err)

	IntSummarizeActions(mm)

	IntAssertAction(mm, Action{Type: ChangeReplicaCountAction, Namespace: "default", DeploymentName: "dummy", ReplicaCt: 1})
	IntAssertAction(mm, Action{Type: VscaleAction, PodName: "pod", ContainerName: "container", CpuRequests: "353m"})
	IntAssertNoActions(mm)

	podlist, err := util.GetReadyPodListForDeployment(a.Clientset, mm.DeploymentName, mm.DeploymentNamespace)
	IntAssertNoError(err)
	IntAssertPodListsEqual(podlist, correctEndPods)

	fmt.Printf("<<< %s passed! >>>\n\n", name)
}

func RunIntegrationTests() {
	IntegrationTest_BasicStable()
	IntegrationTest_BasicVscaleUp()
	IntegrationTest_BasicHscaleUp()
	IntegrationTest_BasicVscaleDown()
	IntegrationTest_BasicHscaleDown()

	fmt.Println("<<< TESTS PASSED SUCCESSFULLY >>>")
}
