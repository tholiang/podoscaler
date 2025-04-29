package testutil

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
	NodeUsages          map[string]int64
	NodeAllocables      map[string]int64
	NodeCapacities      map[string]int64
	DeploymentUtil      int64

	MockGetKubernetesConfig            func(m *MockMetrics) (*rest.Config, error)
	MockGetClientset                   func(m *MockMetrics, config *rest.Config) (*kube_client.Clientset, error)
	MockGetMetricsClientset            func(m *MockMetrics, config *rest.Config) (*metrics_client.Clientset, error)
	MockGetPodListForDeployment        func(m *MockMetrics, clientset kube_client.Interface, deploymentName, namespace string) (*v1.PodList, error)
	MockGetDeploymentUtilAndAlloc      func(m *MockMetrics, clientset kube_client.Interface, metricsClient *metrics_client.Clientset, deploymentName, namespace string, podList *v1.PodList) (int64, int64, error)
	MockGetNodeUsage                   func(m *MockMetrics, metricsClient *metrics_client.Clientset, nodeName string) (int64, error)
	MockGetNodeAllocableAndCapacity    func(m *MockMetrics, clientset kube_client.Interface, nodeName string) (int64, int64, error)
	MockGetLatencyMetrics              func(m *MockMetrics, deployment_name string, percentile float64) (map[string]float64, error)
	MockVScale                         func(m *MockMetrics, clientset kube_client.Interface, podname string, containername string, cpurequests string) error
	MockChangeReplicaCount             func(m *MockMetrics, namespace string, deploymentName string, replicaCt int, clientset kube_client.Interface) error
	MockGetAllDeploymentsFromNamespace func(m *MockMetrics, clientset kube_client.Interface, namespace string) (*appsv1.DeploymentList, error)
	MockDeletePod                      func(m *MockMetrics, clientset kube_client.Interface, podname string, namespace string) error

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
func (m *MockMetrics) GetPodListForDeployment(clientset kube_client.Interface, deploymentName, namespace string) (*v1.PodList, error) {
	return m.MockGetPodListForDeployment(m, clientset, deploymentName, namespace)
}
func (m *MockMetrics) GetDeploymentUtilAndAlloc(clientset kube_client.Interface, metricsClient *metrics_client.Clientset, deploymentName, namespace string, podList *v1.PodList) (int64, int64, error) {
	return m.MockGetDeploymentUtilAndAlloc(m, clientset, metricsClient, deploymentName, namespace, podList)
}
func (m *MockMetrics) GetNodeUsage(metricsClient *metrics_client.Clientset, nodeName string) (int64, error) {
	return m.MockGetNodeUsage(m, metricsClient, nodeName)
}
func (m *MockMetrics) GetNodeAllocableAndCapacity(clientset kube_client.Interface, nodeName string) (int64, int64, error) {
	return m.MockGetNodeAllocableAndCapacity(m, clientset, nodeName)
}
func (m *MockMetrics) GetLatencyMetrics(deployment_name string, percentile float64) (map[string]float64, error) {
	return m.MockGetLatencyMetrics(m, deployment_name, percentile)
}
func (m *MockMetrics) VScale(clientset kube_client.Interface, podname string, containername string, cpurequests string) error {
	return m.MockVScale(m, clientset, podname, containername, cpurequests)
}
func (m *MockMetrics) ChangeReplicaCount(namespace string, deploymentName string, replicaCt int, clientset kube_client.Interface) error {
	return m.MockChangeReplicaCount(m, namespace, deploymentName, replicaCt, clientset)
}

func (m *MockMetrics) GetAllDeploymentsFromNamespace(clientset kube_client.Interface, namespace string) (*appsv1.DeploymentList, error) {
	return m.MockGetAllDeploymentsFromNamespace(m, clientset, namespace)
}

func (m *MockMetrics) DeletePod(clientset kube_client.Interface, podname string, namespace string) error {
	return m.MockDeletePod(m, clientset, podname, namespace)
}
