//go:build autoscaler || autoscalertest
// +build autoscaler autoscalertest

package autoscaler

import (
	"os"

	util "github.com/tholiang/podoscaler/scalers/util"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	kube_client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metrics_client "k8s.io/metrics/pkg/client/clientset/versioned"
)

type DefaultAutoscalerMetrics struct{}

func (m *DefaultAutoscalerMetrics) GetKubernetesConfig() (*rest.Config, error) {
	return rest.InClusterConfig()
}

func (m *DefaultAutoscalerMetrics) GetClientset(config *rest.Config) (*kube_client.Clientset, error) {
	return kubernetes.NewForConfig(config)
}

func (m *DefaultAutoscalerMetrics) GetMetricsClientset(config *rest.Config) (*metrics_client.Clientset, error) {
	return metrics_client.NewForConfig(config)
}

func (m *DefaultAutoscalerMetrics) GetNodeList(clientset kube_client.Interface) (*v1.NodeList, error) {
	return util.GetNodeList(clientset)
}

func (m *DefaultAutoscalerMetrics) GetUnschedulablePodListForDeployment(clientset kube_client.Interface, deploymentName, namespace string) ([]v1.Pod, error) {
	return util.GetUnschedulablePodListForDeployment(clientset, deploymentName, namespace)
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

func (m *DefaultAutoscalerMetrics) GetLatencyMetrics(clientset kube_client.Interface) (map[string]float64, error) {
	lb_name, err := util.GetLoadBalancerName(clientset, os.Getenv("AUTOSCALE_NAMESPACE"), os.Getenv("AUTOSCALE_LB"))
	if err != nil {
		return nil, err
	}
	return util.GetLatencyCloudwatch(lb_name)
}

func (m *DefaultAutoscalerMetrics) VScale(clientset kube_client.Interface, podname string, containername string, cpurequests string, namespace string) error {
	return util.VScale(clientset, podname, containername, cpurequests, namespace)
}

func (m *DefaultAutoscalerMetrics) PatchDeploymentReqs(clientset kube_client.Interface, deploymentName string, containeridx int, cpurequests string, namespace string) error {
	return util.PatchDeploymentReqs(clientset, deploymentName, containeridx, cpurequests, namespace)
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

func (m *DefaultAutoscalerMetrics) GetReadyPodListForDeployment(clientset kube_client.Interface, deploymentName, namespace string) ([]v1.Pod, error) {
	return util.GetReadyPodListForDeployment(clientset, deploymentName, namespace)
}
