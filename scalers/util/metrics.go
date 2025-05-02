package util

import (
	"context"
	"fmt"

	// "math"
	// "slices"
	// "strings"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	kube_client "k8s.io/client-go/kubernetes"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metrics_client "k8s.io/metrics/pkg/client/clientset/versioned"
)

const AUTOSCALE_LABEL = "vecter=true"

func getPodMetricsListForDeployment(clientset kube_client.Interface, metricsClient *metrics_client.Clientset, deploymentName, namespace string) (*v1beta1.PodMetricsList, error) {
	ctx := context.TODO()

	// Get the Deployment
	deployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	// List Pods from metric server using the deployment's label selector
	labelSelector := labels.Set(deployment.Spec.Selector.MatchLabels).AsSelector().String()

	return metricsClient.MetricsV1beta1().PodMetricses(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
}

func GetNodeList(clientset kube_client.Interface) (*v1.NodeList, error) {
	ctx := context.TODO()

	return clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
}

func GetReadyPodListForDeployment(clientset kube_client.Interface, deploymentName, namespace string) ([]v1.Pod, error) {
	ctx := context.TODO()

	// Get the Deployment
	deployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	// List Pods from metric server using the deployment's label selector
	labelSelector := labels.Set(deployment.Spec.Selector.MatchLabels).AsSelector().String()

	podlistobj, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return []v1.Pod{}, err
	}

	podlist := []v1.Pod{}
	for _, poddata := range podlistobj.Items {
		for _, cond := range poddata.Status.Conditions {
			if cond.Type == v1.PodReady {
				podlist = append(podlist, poddata)
				break
			}
		}
	}

	return podlist, nil
}

// returns total utilization, allocation, and # of pods in the deployment
func GetDeploymentUtilAndAlloc(clientset kube_client.Interface, metricsClient *metrics_client.Clientset, deploymentName, namespace string, podList []v1.Pod) (int64, int64, error) {
	podMetricsList, err := getPodMetricsListForDeployment(clientset, metricsClient, deploymentName, namespace)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get podMetricsList: %w", err)
	}

	utilMilli := int64(0)
	for _, podMetrics := range podMetricsList.Items {
		idx := 0
		if podMetrics.Containers[0].Name == "linkerd-proxy" {
			idx = 1
		}
		container := podMetrics.Containers[idx] // TODO: handle multiple containers
		alloc := container.Usage.Cpu().MilliValue()
		utilMilli += alloc
	}
	allocMilli := int64(0)
	for _, pod := range podList {
		idx := 0
		if pod.Spec.Containers[0].Name == "linkerd-proxy" {
			idx = 1
		}
		allocMilli += pod.Spec.Containers[idx].Resources.Requests.Cpu().MilliValue() // TODO: handle multiple containers
	}

	return utilMilli, allocMilli, nil
}

func GetNodeUsage(metricsClient *metrics_client.Clientset, nodeName string) (int64, error) {
	metricsNode, err := metricsClient.MetricsV1beta1().NodeMetricses().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		return 0, fmt.Errorf("failed to get node metrics: %w", err)
	}
	usage := metricsNode.Usage.Cpu().MilliValue()

	return usage, nil
}

func GetNodeAllocableAndCapacity(clientset kube_client.Interface, nodeName string) (int64, int64, error) {
	node, err := clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get node: %w", err)
	}

	capacity := node.Status.Capacity.Cpu().MilliValue()
	allocatable := node.Status.Allocatable.Cpu().MilliValue()

	return allocatable, capacity, nil
}

func GetControlledDeployments(clientset kube_client.Interface) (*appsv1.DeploymentList, error) {
	deployments, err := clientset.AppsV1().Deployments("").List(context.TODO(), metav1.ListOptions{LabelSelector: AUTOSCALE_LABEL})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployments: %w", err)
	}
	return deployments, nil
}

/*	legacy

func GetAllDeploymentsFromNamespace(clientset kube_client.Interface, namespace string) (*appsv1.DeploymentList, error) {
	deployments, err := clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployments: %w", err)
	}
	return deployments, nil
}

*/
