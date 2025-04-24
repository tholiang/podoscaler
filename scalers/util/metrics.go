package util

import (
	"context"
	"fmt"
	// "math"
	// "slices"
	// "strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	kube_client "k8s.io/client-go/kubernetes"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metrics_client "k8s.io/metrics/pkg/client/clientset/versioned"
)

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

func GetPodListForDeployment(clientset kube_client.Interface, deploymentName, namespace string) (*v1.PodList, error) {
	ctx := context.TODO()

	// Get the Deployment
	deployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	// List Pods from metric server using the deployment's label selector
	labelSelector := labels.Set(deployment.Spec.Selector.MatchLabels).AsSelector().String()

	return clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
}

// returns total utilization, allocation, and # of pods in the deployment
func GetDeploymentUtilAndAlloc(clientset kube_client.Interface, metricsClient *metrics_client.Clientset, deploymentName, namespace string, podList *v1.PodList) (int64, int64, error) {
	podMetricsList, err := getPodMetricsListForDeployment(clientset, metricsClient, deploymentName, namespace)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get podMetricsList: %w", err)
	}

	utilMilli := int64(0)
	for _, podMetrics := range podMetricsList.Items {
		container := podMetrics.Containers[0] // TODO: handle multiple containers
		alloc := container.Usage.Cpu().MilliValue()
		utilMilli += alloc
	}
	allocMilli := int64(0)
	for _, pod := range podList.Items {
		allocMilli += pod.Spec.Containers[0].Resources.Requests.Cpu().MilliValue() // TODO: handle multiple containers
	}

	return utilMilli, allocMilli, nil
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