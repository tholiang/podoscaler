package autoscaler

import (
	util "github.com/tholiang/podoscaler/scalers/util"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	kube_client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metrics_client "k8s.io/metrics/pkg/client/clientset/versioned"
)

type DefaultAutoscalerMetrics struct {
}

func (m *DefaultAutoscalerMetrics) GetKubernetesConfig() (*rest.Config, error) {
	return rest.InClusterConfig()
}

func (m *DefaultAutoscalerMetrics) GetClientset(config *rest.Config) (*kube_client.Clientset, error) {
	return kubernetes.NewForConfig(config)
}

func (m *DefaultAutoscalerMetrics) GetMetricsClientset(config *rest.Config) (*metrics_client.Clientset, error) {
	return metrics_client.NewForConfig(config)
}

func (m *DefaultAutoscalerMetrics) GetPodListForDeployment(clientset kube_client.Interface, deploymentName, namespace string) ([]v1.Pod, error) {
	return util.GetReadyPodListForDeployment(clientset, deploymentName, namespace)
}

func (m *DefaultAutoscalerMetrics) GetDeploymentUtilAndAlloc(clientset kube_client.Interface, metricsClient *metrics_client.Clientset, deploymentName, namespace string, podList []v1.Pod) (int64, int64, error) {
	return util.GetDeploymentUtilAndAlloc(clientset, metricsClient, deploymentName, namespace, podList)
}

func (m *DefaultAutoscalerMetrics) GetNodeUsage(metricsClient *metrics_client.Clientset, nodeName string) (int64, error) {
	return util.GetNodeUsage(metricsClient, nodeName)
}

func (m *DefaultAutoscalerMetrics) GetNodeAllocableAndCapacity(clientset kube_client.Interface, nodeName string) (int64, int64, error) {
	return util.GetNodeAllocableAndCapacity(clientset, nodeName)
}

func (m *DefaultAutoscalerMetrics) GetLatencyMetrics(deployment_name string, percentile float64) (map[string]float64, error) {
	return util.GetLatencyMetrics(deployment_name, percentile)
}

func (m *DefaultAutoscalerMetrics) VScale(clientset kube_client.Interface, podname string, containername string, cpurequests string) error {
	return util.VScale(clientset, podname, containername, cpurequests)
}

func (m *DefaultAutoscalerMetrics) ChangeReplicaCount(namespace string, deploymentName string, replicaCt int, clientset kube_client.Interface) error {
	return util.ChangeReplicaCount(namespace, deploymentName, replicaCt, clientset)
}

func (m *DefaultAutoscalerMetrics) GetControlledDeployments(clientset kube_client.Interface) (*appsv1.DeploymentList, error) {
	return util.GetControlledDeployments(clientset)
}

func (m *DefaultAutoscalerMetrics) DeletePod(clientset kube_client.Interface, podname string, namespace string) error {
	return util.DeletePod(clientset, podname, namespace)
}
