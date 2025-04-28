package util

import (
	"fmt"
	"slices"
	"strconv"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	kube_client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metrics_client "k8s.io/metrics/pkg/client/clientset/versioned"
)

/* random helpers */
func AssertNoError(e error, t *testing.T) {
	if e != nil {
		t.Errorf("error: %s", e.Error())
	}
}

func GetStringIntMapKeys(m map[string]int) []string {
	keys := make([]string, len(m))

	i := 0
	for k := range m {
		keys[i] = k
		i++
	}

	return keys
}

func AssertStringIntMapsEqual(m1 map[string]int, m2 map[string]int, t *testing.T) {
	testkeys := GetStringIntMapKeys(m1)
	correctkeys := GetStringIntMapKeys(m2)
	if !slices.Equal(testkeys, correctkeys) {
		t.Errorf("map key mismatch")
	}

	for _, k := range testkeys {
		if m1[k] != m2[k] {
			t.Errorf("map mismatch at %s, expected %d, got %d", k, m2[k], m1[k])
		}
	}
}

func GetPodListKeys(m MockPodList) []string {
	keys := make([]string, len(m))

	i := 0
	for k := range m {
		keys[i] = k
		i++
	}

	return keys
}

func AssertPodListsEqual(l1 MockPodList, l2 MockPodList, t *testing.T) {
	testkeys := GetPodListKeys(l1)
	correctkeys := GetPodListKeys(l2)
	if !slices.Equal(testkeys, correctkeys) {
		t.Errorf("podlist key mismatch")
	}

	for _, k := range testkeys {
		if l1[k].PodName != l2[k].PodName {
			t.Errorf("pod mismatch at %s, expected pod name %s, got %s", k, l2[k].PodName, l1[k].PodName)
		}
		if l1[k].NodeName != l2[k].NodeName {
			t.Errorf("pod mismatch at %s, expected node name %s, got %s", k, l2[k].NodeName, l1[k].NodeName)
		}
		if l1[k].ContainerName != l2[k].ContainerName {
			t.Errorf("pod mismatch at %s, expected container name %s, got %s", k, l2[k].ContainerName, l1[k].ContainerName)
		}
		if l1[k].CpuRequests != l2[k].CpuRequests {
			t.Errorf("pod mismatch at %s, expected %d cpu requests, got %d", k, l2[k].CpuRequests, l1[k].CpuRequests)
		}
	}
}

/* generalizable mock metrics setup */
const MOCK_DEPLOYMENT_NAME = "testapp"
const MOCK_DEPLOYMENT_NAMESPACE = "default"
const MOCK_MAPS = 500              // in millicpus
const MOCK_LATENCY_THRESHOLD = 100 // in milliseconds

/* mock util */
func MakePod(podName string, nodeName string, containerName string, cpuRequests int64) v1.Pod {
	pod := v1.Pod{}
	pod.Name = podName
	pod.Spec.NodeName = nodeName
	pod.Spec.Containers = []v1.Container{
		{
			Name: containerName,
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					"cpu": *resource.NewMilliQuantity(cpuRequests, resource.DecimalSI),
				},
			},
		},
	}

	return pod
}

func MakeDeployment(deploymentName string, namespace string, replicas int32) appsv1.Deployment {
	deployment := appsv1.Deployment{}
	deployment.Name = deploymentName
	deployment.Namespace = namespace
	deployment.Spec.Replicas = &replicas
	return deployment
}

func MockPodListToPodList(mock MockPodList) *v1.PodList {
	podlist := new(v1.PodList) // bad practice probably
	for _, v := range mock {
		podlist.Items = append(podlist.Items, MakePod(v.PodName, v.NodeName, v.ContainerName, v.CpuRequests))
	}
	return podlist
}

func GetDeploymentAlloc(pods MockPodList) int64 {
	var alloc int64 = 0
	for _, v := range pods {
		alloc += v.CpuRequests
	}

	return alloc
}

/* general (shared) mock functions */
func MockConfig(m *MockMetrics) (*rest.Config, error) {
	return new(rest.Config), nil
}

func MockClientset(m *MockMetrics, config *rest.Config) (*kube_client.Clientset, error) {
	return new(kube_client.Clientset), nil
}

func MockMetricsClientset(m *MockMetrics, config *rest.Config) (*metrics_client.Clientset, error) {
	return new(metrics_client.Clientset), nil
}

func MockAllDeploymentsFromNamespace(m *MockMetrics, clientset kube_client.Interface, namespace string) (*appsv1.DeploymentList, error) {
	deploymentList := new(appsv1.DeploymentList)
	deploymentList.Items = []appsv1.Deployment{
		MakeDeployment(m.DeploymentName, m.DeploymentNamespace, 1),
	}
	return deploymentList, nil
}

func MockPodListForDeployment(m *MockMetrics, clientset kube_client.Interface, deploymentName, namespace string) (*v1.PodList, error) {
	return MockPodListToPodList(m.Pods), nil
}

func MockDeploymentUtilAndAlloc(m *MockMetrics, clientset kube_client.Interface, metricsClient *metrics_client.Clientset, deploymentName, namespace string, podList *v1.PodList) (int64, int64, error) {
	return m.DeploymentUtil, GetDeploymentAlloc(m.Pods), nil
}

func MockNodeAllocableAndCapacity(m *MockMetrics, clientset kube_client.Interface, nodeName string) (int64, int64, error) {
	alloc, ok := m.NodeAllocables[nodeName]
	if !ok {
		return 0, 0, fmt.Errorf("couldn't find allocable for node %s", nodeName)
	}
	cap, ok := m.NodeCapacities[nodeName]
	if !ok {
		return 0, 0, fmt.Errorf("couldn't find capacity for node %s", nodeName)
	}

	return alloc, cap, nil
}

func MockLatencyMetrics(m *MockMetrics, deployment_name string, percentile float64) (map[string]float64, error) {
	metrics := map[string]float64{
		m.DeploymentName: m.Latency,
	}
	return metrics, nil
}

func MockVScale(m *MockMetrics, clientset kube_client.Interface, podname string, containername string, cpurequests string) error {
	data, ok := m.Pods[podname]
	if !ok {
		return fmt.Errorf("failed to get pod with name: %s", podname)
	}
	if data.ContainerName != containername {
		return fmt.Errorf("found incorrect container name for pod %s, expected %s, got %s", podname, data.ContainerName, containername)
	}

	var err error
	data.CpuRequests, err = strconv.ParseInt(cpurequests[:len(cpurequests)-1], 10, 64) // assume format it "[num]m" for millis
	if err != nil {
		return fmt.Errorf("failed to parse cpurequests string: %s", err.Error())
	}

	m.Pods[podname] = data

	return nil
}

func MockChangeReplicaCount(m *MockMetrics, namespace string, deploymentName string, replicaCt int, clientset kube_client.Interface) error {
	numpods := len(m.Pods)
	if numpods < replicaCt {
		for i := numpods + 1; i <= replicaCt; i++ {
			podname := fmt.Sprintf("pod%d", i)
			m.Pods[podname] = PodData{PodName: podname, NodeName: "node2", ContainerName: "container", CpuRequests: 300}
		}
	} else if numpods > replicaCt {
		for i := numpods; i > replicaCt; i-- {
			podname := fmt.Sprintf("pod%d", i)
			delete(m.Pods, podname)
		}
	}

	return nil
}

func MockDeletePod(m *MockMetrics, clientset kube_client.Interface, podname string, namespace string) error {
	_, ok := m.Pods[podname]
	if !ok {
		return fmt.Errorf("failed to delete pod, no pod found with name: %s", podname)
	}

	delete(m.Pods, podname)
	return nil
}

func CreateSimpleMockMetrics() *MockMetrics {
	mm := new(MockMetrics)
	mm.MockGetKubernetesConfig = MockConfig
	mm.MockGetClientset = MockClientset
	mm.MockGetMetricsClientset = MockMetricsClientset
	mm.MockGetAllDeploymentsFromNamespace = MockAllDeploymentsFromNamespace
	mm.MockGetPodListForDeployment = MockPodListForDeployment
	mm.MockGetDeploymentUtilAndAlloc = MockDeploymentUtilAndAlloc
	mm.MockGetNodeAllocableAndCapacity = MockNodeAllocableAndCapacity
	mm.MockGetLatencyMetrics = MockLatencyMetrics
	mm.MockVScale = MockVScale
	mm.MockChangeReplicaCount = MockChangeReplicaCount
	mm.MockDeletePod = MockDeletePod

	// default values
	mm.DeploymentName = MOCK_DEPLOYMENT_NAME
	mm.DeploymentNamespace = MOCK_DEPLOYMENT_NAMESPACE
	mm.Pods = map[string]PodData{
		"pod1": {"pod1", "node1", "container", 300},
		"pod2": {"pod2", "node1", "container", 300},
		"pod3": {"pod3", "node2", "container", 300},
	}
	mm.Latency = MOCK_LATENCY_THRESHOLD * 0.95
	mm.NodeAllocables = map[string]int64{
		"node1": 400,
		"node2": 700,
	}
	mm.NodeCapacities = map[string]int64{
		"node1": 1000,
		"node2": 1000,
	}
	mm.DeploymentUtil = 600

	return mm
}
