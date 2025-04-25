package util

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	kube_client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metrics_client "k8s.io/metrics/pkg/client/clientset/versioned"
)

/* random helpers */
func GetStringIntMapKeys(m map[string]int) ([]string) {
	keys := make([]string, len(m))

	i := 0
	for k := range m {
		keys[i] = k
		i++
	}

	return keys
}

/* generalizable mock metrics setup */
const MOCK_DEPLOYMENT_NAME = "testapp"
const MOCK_DEPLOYMENT_NAMESPACE = "default"
const MOCK_MAPS = 500              // in millicpus
const MOCK_LATENCY_THRESHOLD = 100 // in milliseconds

type MockMetrics struct {
	MockGetKubernetesConfig         func() (*rest.Config, error)
	MockGetClientset                func(config *rest.Config) (*kube_client.Clientset, error)
	MockGetMetricsClientset         func(config *rest.Config) (*metrics_client.Clientset, error)
	MockGetPodListForDeployment     func(clientset kube_client.Interface, deploymentName, namespace string) (*v1.PodList, error)
	MockGetDeploymentUtilAndAlloc   func(clientset kube_client.Interface, metricsClient *metrics_client.Clientset, deploymentName, namespace string, podList *v1.PodList) (int64, int64, error)
	MockGetNodeAllocableAndCapacity func(clientset kube_client.Interface, nodeName string) (int64, int64, error)
	MockGetLatencyMetrics           func(deployment_name string, percentile float64) (map[string]float64, error)
	MockVScale                      func(clientset kube_client.Interface, podname string, containername string, cpurequests string) error
	MockChangeReplicaCount          func(namespace string, deploymentName string, replicaCt int, clientset kube_client.Interface) error
}

func (m *MockMetrics) GetKubernetesConfig() (*rest.Config, error) {
	return m.MockGetKubernetesConfig()
}
func (m *MockMetrics) GetClientset(config *rest.Config) (*kube_client.Clientset, error) {
	return m.MockGetClientset(config)
}
func (m *MockMetrics) GetMetricsClientset(config *rest.Config) (*metrics_client.Clientset, error) {
	return m.MockGetMetricsClientset(config)
}
func (m *MockMetrics) GetPodListForDeployment(clientset kube_client.Interface, deploymentName, namespace string) (*v1.PodList, error) {
	return m.MockGetPodListForDeployment(clientset, deploymentName, namespace)
}
func (m *MockMetrics) GetDeploymentUtilAndAlloc(clientset kube_client.Interface, metricsClient *metrics_client.Clientset, deploymentName, namespace string, podList *v1.PodList) (int64, int64, error) {
	return m.MockGetDeploymentUtilAndAlloc(clientset, metricsClient, deploymentName, namespace, podList)
}
func (m *MockMetrics) GetNodeAllocableAndCapacity(clientset kube_client.Interface, nodeName string) (int64, int64, error) {
	return m.MockGetNodeAllocableAndCapacity(clientset, nodeName)
}
func (m *MockMetrics) GetLatencyMetrics(deployment_name string, percentile float64) (map[string]float64, error) {
	return m.MockGetLatencyMetrics(deployment_name, percentile)
}
func (m *MockMetrics) VScale(clientset kube_client.Interface, podname string, containername string, cpurequests string) error {
	return m.MockVScale(clientset, podname, containername, cpurequests)
}
func (m *MockMetrics) ChangeReplicaCount(namespace string, deploymentName string, replicaCt int, clientset kube_client.Interface) error {
	return m.MockChangeReplicaCount(namespace, deploymentName, replicaCt, clientset)
}

/* general (shared) mock functions */
func FakeConfig() (*rest.Config, error) {
	return new(rest.Config), nil
}

func FakeClientset(config *rest.Config) (*kube_client.Clientset, error) {
	return new(kube_client.Clientset), nil
}

func FakeMetricsClientset(config *rest.Config) (*metrics_client.Clientset, error) {
	return new(metrics_client.Clientset), nil
}

func makePod(podName string, nodeName string, cpuRequests int64) v1.Pod {
	pod := v1.Pod{}
	pod.Name = podName
	pod.Spec.NodeName = nodeName
	pod.Spec.Containers = []v1.Container{
		v1.Container{
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					"cpu": *resource.NewMilliQuantity(cpuRequests, resource.DecimalSI),
				},
			},
		},
	}

	return pod
}

func SimplePodListForDeployment(clientset kube_client.Interface, deploymentName, namespace string) (*v1.PodList, error) {
	// return pod list of size 3
	podlist := new(v1.PodList)
	podlist.Items = []v1.Pod{
		makePod("pod1", "node1", 300),
		makePod("pod2", "node1", 300),
		makePod("pod3", "node2", 300),
	}
	return podlist, nil
}

func SimpleFineDeploymentUtilAndAlloc(clientset kube_client.Interface, metricsClient *metrics_client.Clientset, deploymentName, namespace string, podList *v1.PodList) (int64, int64, error) {
	return 299, 300, nil
}
func SimpleOverDeploymentUtilAndAlloc(clientset kube_client.Interface, metricsClient *metrics_client.Clientset, deploymentName, namespace string, podList *v1.PodList) (int64, int64, error) {
	return 320, 300, nil
}
func SimpleUnderDeploymentUtilAndAlloc(clientset kube_client.Interface, metricsClient *metrics_client.Clientset, deploymentName, namespace string, podList *v1.PodList) (int64, int64, error) {
	return 100, 300, nil
}
func SimpleUncongestedNodeAllocableAndCapacity(clientset kube_client.Interface, nodeName string) (int64, int64, error) {
	return 800, 1000, nil
}
func SimpleCongestedNodeAllocableAndCapacity(clientset kube_client.Interface, nodeName string) (int64, int64, error) {
	return 100, 1000, nil
}
func SimpleGoodLatencyMetrics(deployment_name string, percentile float64) (map[string]float64, error) {
	metrics := map[string]float64{
		MOCK_DEPLOYMENT_NAME: MOCK_LATENCY_THRESHOLD * 0.95,
	}
	return metrics, nil
}
func SimpleOverLatencyMetrics(deployment_name string, percentile float64) (map[string]float64, error) {
	metrics := map[string]float64{
		MOCK_DEPLOYMENT_NAME: MOCK_LATENCY_THRESHOLD * 2,
	}
	return metrics, nil
}
func SimpleUnderLatencyMetrics(deployment_name string, percentile float64) (map[string]float64, error) {
	metrics := map[string]float64{
		MOCK_DEPLOYMENT_NAME: MOCK_LATENCY_THRESHOLD * 0.5,
	}
	return metrics, nil
}
func DummyVScale(clientset kube_client.Interface, podname string, containername string, cpurequests string) error {
	return nil
}
func DummyChangeReplicaCount(namespace string, deploymentName string, replicaCt int, clientset kube_client.Interface) error {
	return nil
}

func CreateSimpleMockMetrics() *MockMetrics {
	mm := new(MockMetrics)
	mm.MockGetKubernetesConfig = FakeConfig
	mm.MockGetClientset = FakeClientset
	mm.MockGetMetricsClientset = FakeMetricsClientset
	mm.MockGetPodListForDeployment = SimplePodListForDeployment
	mm.MockGetDeploymentUtilAndAlloc = SimpleFineDeploymentUtilAndAlloc
	mm.MockGetNodeAllocableAndCapacity = SimpleUncongestedNodeAllocableAndCapacity
	mm.MockGetLatencyMetrics = SimpleGoodLatencyMetrics
	mm.MockVScale = DummyVScale
	mm.MockChangeReplicaCount = DummyChangeReplicaCount

	return mm
}
