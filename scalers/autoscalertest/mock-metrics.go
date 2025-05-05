//go:build autoscalertest
// +build autoscalertest

package autoscalertest

import (
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	kube_client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metrics_client "k8s.io/metrics/pkg/client/clientset/versioned"
)

type PodData struct {
	PodName       string
	NodeName      string
	ContainerName string
	CpuRequests   int64
}
type MockPodList map[string]PodData

type ActionType string

const (
	VscaleAction             ActionType = "vscale"
	ChangeReplicaCountAction ActionType = "change replica"
	DeletePodAction          ActionType = "delete"
)

type Action struct {
	Type           ActionType
	Namespace      string // change replica, delete
	DeploymentName string // change replica
	ReplicaCt      int    // change replica
	PodName        string // vscale, delete
	ContainerName  string // vscale
	CpuRequests    string // vscale
}

type MockMetrics struct {
	DeploymentName      string
	DeploymentNamespace string
	Pods                MockPodList
	Latency             float64
	RelNodeUsages       map[string]float64
	NodeAllocables      map[string]int64
	NodeCapacities      map[string]int64
	RelDeploymentUtil   float64

	MockGetKubernetesConfig          func(m *MockMetrics) (*rest.Config, error)
	MockGetClientset                 func(m *MockMetrics, config *rest.Config) (*kube_client.Clientset, error)
	MockGetMetricsClientset          func(m *MockMetrics, config *rest.Config) (*metrics_client.Clientset, error)
	MockGetNodeList                  func(m *MockMetrics, clientset kube_client.Interface) (*v1.NodeList, error)
	MockGetReadyPodListForDeployment func(m *MockMetrics, clientset kube_client.Interface, deploymentName, namespace string) ([]v1.Pod, error)
	MockGetDeploymentUtilAndAlloc    func(m *MockMetrics, clientset kube_client.Interface, metricsClient *metrics_client.Clientset, deploymentName, namespace string, podList []v1.Pod) (int64, int64, error)
	MockGetNodeUsage                 func(m *MockMetrics, metricsClient *metrics_client.Clientset, nodeName string) (int64, error)
	MockGetNodeAllocableAndCapacity  func(m *MockMetrics, clientset kube_client.Interface, nodeName string) (int64, int64, error)
	MockGetLatencyMetrics            func(m *MockMetrics, clientset kube_client.Interface) (map[string]float64, error)
	MockVScale                       func(m *MockMetrics, clientset kube_client.Interface, podname string, containername string, cpurequests string, namespace string) error
	MockChangeReplicaCount           func(m *MockMetrics, namespace string, deploymentName string, replicaCt int, clientset kube_client.Interface) error
	MockGetControlledDeployments     func(m *MockMetrics, clientset kube_client.Interface) (*appsv1.DeploymentList, error)
	MockDeletePod                    func(m *MockMetrics, clientset kube_client.Interface, podname string, namespace string) error

	Actions []Action // log in MockVScale, MockChangeReplicaCount, MockDeletePod implementations
}

func (m *MockMetrics) GetKubernetesConfig() (*rest.Config, error) {
	return m.MockGetKubernetesConfig(m)
}
func (m *MockMetrics) GetClientset(config *rest.Config) (*kube_client.Clientset, error) {
	return m.MockGetClientset(m, config)
}
func (m *MockMetrics) GetMetricsClientset(config *rest.Config) (*metrics_client.Clientset, error) {
	return m.MockGetMetricsClientset(m, config)
}
func (m *MockMetrics) GetNodeList(clientset kube_client.Interface) (*v1.NodeList, error) {
	return m.MockGetNodeList(m, clientset)
}
func (m *MockMetrics) GetReadyPodListForDeployment(clientset kube_client.Interface, deploymentName, namespace string) ([]v1.Pod, error) {
	return m.MockGetReadyPodListForDeployment(m, clientset, deploymentName, namespace)
}
func (m *MockMetrics) GetDeploymentUtilAndAlloc(clientset kube_client.Interface, metricsClient *metrics_client.Clientset, deploymentName, namespace string, podList []v1.Pod) (int64, int64, error) {
	return m.MockGetDeploymentUtilAndAlloc(m, clientset, metricsClient, deploymentName, namespace, podList)
}
func (m *MockMetrics) GetNodeUsage(metricsClient *metrics_client.Clientset, nodeName string) (int64, error) {
	return m.MockGetNodeUsage(m, metricsClient, nodeName)
}
func (m *MockMetrics) GetNodeAllocableAndCapacity(clientset kube_client.Interface, nodeName string) (int64, int64, error) {
	return m.MockGetNodeAllocableAndCapacity(m, clientset, nodeName)
}
func (m *MockMetrics) GetLatencyMetrics(clientset kube_client.Interface) (map[string]float64, error) {
	return m.MockGetLatencyMetrics(m, clientset)
}
func (m *MockMetrics) VScale(clientset kube_client.Interface, podname string, containername string, cpurequests string, namespace string) error {
	return m.MockVScale(m, clientset, podname, containername, cpurequests, namespace)
}
func (m *MockMetrics) ChangeReplicaCount(namespace string, deploymentName string, replicaCt int, clientset kube_client.Interface) error {
	return m.MockChangeReplicaCount(m, namespace, deploymentName, replicaCt, clientset)
}

func (m *MockMetrics) GetControlledDeployments(clientset kube_client.Interface) (*appsv1.DeploymentList, error) {
	return m.MockGetControlledDeployments(m, clientset)
}

func (m *MockMetrics) DeletePod(clientset kube_client.Interface, podname string, namespace string) error {
	return m.MockDeletePod(m, clientset, podname, namespace)
}
