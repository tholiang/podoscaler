package autoscalertest

import (
	"fmt"

	"github.com/tholiang/podoscaler/scalers/util"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	kube_client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metrics_client "k8s.io/metrics/pkg/client/clientset/versioned"
)

/* random helpers */
func IntAssertNoError(e error) {
	if e != nil {
		panic(fmt.Sprintf("error: %s", e.Error()))
	}
}

func IntAssertIntsEqual(i1 int, i2 int) {
	if i1 != i2 {
		panic(fmt.Sprintf("unequal ints, expected %d, got %d", i1, i2))
	}
}

func IntSummarizeActions(mm *MockMetrics) {
	vscales := 0
	changereplicas := 0
	deletes := 0
	for _, d := range mm.Actions {
		if d.Type == VscaleAction {
			vscales++
		} else if d.Type == ChangeReplicaCountAction {
			changereplicas++
		} else if d.Type == DeletePodAction {
			deletes++
		}
	}

	fmt.Println("<<< actions performed: ")
	fmt.Printf("%d vscales\n", vscales)
	fmt.Printf("%d replica count changes\n", changereplicas)
	fmt.Printf("%d deletes\n", deletes)
	fmt.Println(">>>")
}

func IntAssertAction(mm *MockMetrics, a Action) {
	if len(mm.Actions) == 0 {
		panic(fmt.Sprintf("no actions found, expected %s", a.Type))
	}

	// compare first action to given and pop
	first_action := mm.Actions[0]
	if first_action.Type != a.Type {
		panic(fmt.Sprintf("incorrect action type, expected %s, got %s", a.Type, first_action.Type))
	}

	if a.Type == VscaleAction {
		// ignore because order may be arbitrary
		// if a.PodName != first_action.PodName {
		// 	panic(fmt.Sprintf("incorrect pod name for vscale, expected %s, got %s", a.PodName, first_action.PodName))
		// }
		// if a.ContainerName != first_action.ContainerName {
		// 	panic(fmt.Sprintf("incorrect container name for vscale, expected %s, got %s", a.ContainerName, first_action.ContainerName))
		// }
		if a.CpuRequests != first_action.CpuRequests {
			panic(fmt.Sprintf("incorrect cpu request for vscale, expected %s, got %s", a.CpuRequests, first_action.CpuRequests))
		}
	} else if a.Type == ChangeReplicaCountAction {
		if a.Namespace != first_action.Namespace {
			panic(fmt.Sprintf("incorrect namespace for change replica, expected %s, got %s", a.Namespace, first_action.Namespace))
		}
		if a.DeploymentName != first_action.DeploymentName {
			panic(fmt.Sprintf("incorrect deployment name for change replica, expected %s, got %s", a.DeploymentName, first_action.DeploymentName))
		}
		if a.ReplicaCt != first_action.ReplicaCt {
			panic(fmt.Sprintf("incorrect replica count for change replica, expected %d, got %d", a.ReplicaCt, first_action.ReplicaCt))
		}
	} else {
		if a.DeploymentName != first_action.DeploymentName {
			panic(fmt.Sprintf("incorrect deployment name for delete, expected %s, got %s", a.DeploymentName, first_action.DeploymentName))
		}
		// if a.PodName != first_action.PodName {
		// 	panic(fmt.Sprintf("incorrect pod name for delete, expected %s, got %s", a.PodName, first_action.PodName))
		// }
	}

	mm.Actions = mm.Actions[1:]
}

func IntAssertNoActions(mm *MockMetrics) {
	if len(mm.Actions) > 0 {
		for _, a := range mm.Actions {
			fmt.Printf("unexpected action: %s\n", a.Type)
		}
		panic(fmt.Sprintf("found %d unexpected actions", len(mm.Actions)))
	}
}

func IntAssertPodListsEqual(podlist []v1.Pod, correctEndPods []PodData) {
	IntAssertIntsEqual(len(podlist), len(correctEndPods))

	for i, d := range correctEndPods {
		// should be all the same size so order doesn't matter
		correctCpu := d.CpuRequests
		trueCpu := podlist[i].Spec.Containers[0].Resources.Requests.Cpu().MilliValue()
		IntAssertIntsEqual(int(correctCpu), int(trueCpu))
	}
}

/* mock functions for integration testing */
func IntMockConfig(m *MockMetrics) (*rest.Config, error) {
	return rest.InClusterConfig()
}

func IntMockClientset(m *MockMetrics, config *rest.Config) (*kube_client.Clientset, error) {
	return kubernetes.NewForConfig(config)
}

func IntMockMetricsClientset(m *MockMetrics, config *rest.Config) (*metrics_client.Clientset, error) {
	return metrics_client.NewForConfig(config)
}

func IntMockControlledDeployments(m *MockMetrics, clientset kube_client.Interface) (*appsv1.DeploymentList, error) {
	return util.GetControlledDeployments(clientset)
}

func IntMockReadyPodListForDeployment(m *MockMetrics, clientset kube_client.Interface, deploymentName, namespace string) ([]v1.Pod, error) {
	return util.GetReadyPodListForDeployment(clientset, deploymentName, namespace)
}

func IntMockDeploymentUtilAndAlloc(m *MockMetrics, clientset kube_client.Interface, metricsClient *metrics_client.Clientset, deploymentName, namespace string, podList []v1.Pod) (int64, int64, error) {
	_, alloc, err := util.GetDeploymentUtilAndAlloc(clientset, metricsClient, deploymentName, namespace, podList)
	if err != nil {
		return 0, 0, err
	}
	return int64(m.RelDeploymentUtil * float64(alloc)), alloc, nil
}

func IntMockNodeUsage(m *MockMetrics, metricsClient *metrics_client.Clientset, nodeName string) (int64, error) {
	cap, ok := m.NodeCapacities[nodeName]
	if !ok {
		return 0, fmt.Errorf("node capacities is not set for node %s - sorry this is a workaround", nodeName)
	}
	usage, ok := m.RelNodeUsages[nodeName]
	if !ok {
		return 0, fmt.Errorf("couldn't find usage for node %s", nodeName)
	}

	return int64(usage * float64(cap)), nil
}

func IntMockNodeAllocableAndCapacity(m *MockMetrics, clientset kube_client.Interface, nodeName string) (int64, int64, error) {
	return util.GetNodeAllocableAndCapacity(clientset, nodeName)
}

func IntMockLatencyMetrics(m *MockMetrics, deployment_name string, percentile float64) (map[string]float64, error) {
	metrics := map[string]float64{
		m.DeploymentName: m.Latency,
	}
	return metrics, nil
}

func IntMockVScale(m *MockMetrics, clientset kube_client.Interface, podname string, containername string, cpurequests string, namespace string) error {
	err := util.VScale(clientset, podname, containername, cpurequests, namespace)
	if err != nil {
		return err
	}

	m.Actions = append(m.Actions, Action{Type: VscaleAction, PodName: podname, ContainerName: containername, CpuRequests: cpurequests, Namespace: namespace})
	return nil
}

func IntMockChangeReplicaCount(m *MockMetrics, namespace string, deploymentName string, replicaCt int, clientset kube_client.Interface) error {
	err := util.ChangeReplicaCount(namespace, deploymentName, replicaCt, clientset)
	if err != nil {
		return err
	}

	m.Actions = append(m.Actions, Action{Type: ChangeReplicaCountAction, Namespace: namespace, DeploymentName: deploymentName, ReplicaCt: replicaCt})
	return nil
}

func IntMockDeletePod(m *MockMetrics, clientset kube_client.Interface, podname string, namespace string) error {
	err := util.DeletePod(clientset, podname, namespace)
	if err != nil {
		return err
	}

	m.Actions = append(m.Actions, Action{Type: DeletePodAction, PodName: podname, Namespace: namespace})
	return nil
}

func CreateIntMockMetrics() *MockMetrics {
	mm := new(MockMetrics)
	mm.MockGetKubernetesConfig = IntMockConfig
	mm.MockGetClientset = IntMockClientset
	mm.MockGetMetricsClientset = IntMockMetricsClientset
	mm.MockGetControlledDeployments = IntMockControlledDeployments
	mm.MockGetReadyPodListForDeployment = IntMockReadyPodListForDeployment
	mm.MockGetDeploymentUtilAndAlloc = IntMockDeploymentUtilAndAlloc
	mm.MockGetNodeUsage = IntMockNodeUsage
	mm.MockGetNodeAllocableAndCapacity = IntMockNodeAllocableAndCapacity
	mm.MockGetLatencyMetrics = IntMockLatencyMetrics
	mm.MockVScale = IntMockVScale
	mm.MockChangeReplicaCount = IntMockChangeReplicaCount
	mm.MockDeletePod = IntMockDeletePod

	// default values
	mm.DeploymentName = "dummy"
	mm.DeploymentNamespace = "default"
	mm.Latency = MOCK_LATENCY_THRESHOLD * 0.95
	mm.RelNodeUsages = map[string]float64{
		"minikube": 0.3,
	}
	mm.RelDeploymentUtil = 0.9

	return mm
}
