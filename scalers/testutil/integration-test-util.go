package testutil

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

func IntMockAllDeploymentsFromNamespace(m *MockMetrics, clientset kube_client.Interface, namespace string) (*appsv1.DeploymentList, error) {
	return util.GetAllDeploymentsFromNamespace(clientset, namespace)
}

func IntMockPodListForDeployment(m *MockMetrics, clientset kube_client.Interface, deploymentName, namespace string) (*v1.PodList, error) {
	return util.GetPodListForDeployment(clientset, deploymentName, namespace)
}

func IntMockDeploymentUtilAndAlloc(m *MockMetrics, clientset kube_client.Interface, metricsClient *metrics_client.Clientset, deploymentName, namespace string, podList *v1.PodList) (int64, int64, error) {
	alloc := GetDeploymentAlloc(m.Pods)
	return int64(m.RelDeploymentUtil * float64(alloc)), alloc, nil
}

func IntMockNodeUsage(m *MockMetrics, metricsClient *metrics_client.Clientset, nodeName string) (int64, error) {
	usage, ok := m.NodeUsages[nodeName]
	if !ok {
		return 0, fmt.Errorf("couldn't find usage for node %s", nodeName)
	}

	return usage, nil
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

func IntMockVScale(m *MockMetrics, clientset kube_client.Interface, podname string, containername string, cpurequests string) error {
	err := util.VScale(clientset, podname, containername, cpurequests)
	if err != nil {
		return err
	}

	m.Actions = append(m.Actions, Action{Type: VscaleAction, PodName: podname, ContainerName: containername, CpuRequests: cpurequests})
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
