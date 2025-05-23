//go:build autoscaler || autoscalertest
// +build autoscaler autoscalertest

package autoscaler

import (
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	kube_client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metrics_client "k8s.io/metrics/pkg/client/clientset/versioned"
)

type AutoscalerMetrics interface {
	GetKubernetesConfig() (*rest.Config, error)
	GetClientset(config *rest.Config) (*kube_client.Clientset, error)
	GetMetricsClientset(config *rest.Config) (*metrics_client.Clientset, error)
	GetNodeList(clientset kube_client.Interface) (*v1.NodeList, error)
	GetUnschedulablePodListForDeployment(clientset kube_client.Interface, deploymentName, namespace string) ([]v1.Pod, error)
	GetReadyPodListForDeployment(clientset kube_client.Interface, deploymentName, namespace string) ([]v1.Pod, error)
	GetDeploymentUtilAndAlloc(clientset kube_client.Interface, metricsClient *metrics_client.Clientset, deploymentName, namespace string, podList []v1.Pod) (int64, int64, error)
	GetNodeUsage(metricsClient *metrics_client.Clientset, nodeName string) (int64, error)
	GetNodeAllocableAndCapacity(clientset kube_client.Interface, nodeName string) (int64, int64, error)
	GetLatencyMetrics(client_set kube_client.Interface) (map[string]float64, error)
	VScale(clientset kube_client.Interface, podname string, containername string, cpurequests string, namespace string) error
	PatchDeploymentReqs(clientset kube_client.Interface, deploymentName string, containeridx int, cpurequests string, namespace string) error
	ChangeReplicaCount(namespace string, deploymentName string, replicaCt int, clientset kube_client.Interface) error
	GetControlledDeployments(clientset kube_client.Interface) (*appsv1.DeploymentList, error)
	DeletePod(clientset kube_client.Interface, podname string, namespace string) error
}

type AutoscalerInterface interface {
	Init() error
	RunRound() error
	isSLOViolated() bool
	vScaleTo(millis int64) error
	hScale(idealReplicaCt int) error
}
