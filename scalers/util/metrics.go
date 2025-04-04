package util

import (
	"context"
	"fmt"
	"math"
	
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metrics_client "k8s.io/metrics/pkg/client/clientset/versioned"
	kube_client "k8s.io/client-go/kubernetes"
	"k8s.io/apimachinery/pkg/labels"
)

func GetPodMetrics(clientset *metrics_client.Clientset) *v1beta1.PodMetricsList {
	podMetricsList, err := clientset.MetricsV1beta1().PodMetricses("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}

	return podMetricsList
}

func GetSmallestPodOfDeployment(clientset kube_client.Interface, metricsClient *metrics_client.Clientset, deploymentName, namespace string) (*v1beta1.PodMetrics, error) {
	ctx := context.TODO()

	// Get the Deployment
	deployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	// List Pods using the deployment's label selector
	labelSelector := labels.Set(deployment.Spec.Selector.MatchLabels).AsSelector().String()
	podList, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}
	if len(podList.Items) == 0 {
		return nil, fmt.Errorf("no pods found for deployment %q", deploymentName)
	}

	var smallestPod *v1beta1.PodMetrics
	smallestCPUUsageMilli := int64(math.MaxInt64)

	// Iterate through each pod and get its CPU usage via the metrics API.
	for _, pod := range podList.Items {
		podMetrics, err := metricsClient.MetricsV1beta1().PodMetricses(namespace).Get(ctx, pod.Name, metav1.GetOptions{})
		if err != nil {
			// log.Printf("Warning: could not get metrics for pod %s: %v", pod.Name, err)
			continue
		}
		
		usage := podMetrics.Containers[0].Usage.Cpu().MilliValue()
		if usage < smallestCPUUsageMilli {
			smallestCPUUsageMilli = usage
			smallestPod = podMetrics
		}
	}

	return smallestPod, nil
}

func GetLargestPodOfDeployment(clientset kube_client.Interface, metricsClient *metrics_client.Clientset, deploymentName, namespace string) (*v1beta1.PodMetrics, error) {
	ctx := context.TODO()

	// Get the Deployment
	deployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	// List Pods using the deployment's label selector
	labelSelector := labels.Set(deployment.Spec.Selector.MatchLabels).AsSelector().String()
	podList, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}
	if len(podList.Items) == 0 {
		return nil, fmt.Errorf("no pods found for deployment %q", deploymentName)
	}

	var largestPod *v1beta1.PodMetrics
	largestCPUUsageMilli := int64(math.MinInt64)

	// Iterate through each pod and get its CPU usage via the metrics API.
	for _, pod := range podList.Items {
		podMetrics, err := metricsClient.MetricsV1beta1().PodMetricses(namespace).Get(ctx, pod.Name, metav1.GetOptions{})
		if err != nil {
			// log.Printf("Warning: could not get metrics for pod %s: %v", pod.Name, err)
			continue
		}
		
		usage := podMetrics.Containers[0].Usage.Cpu().MilliValue()
		if usage > largestCPUUsageMilli {
			largestCPUUsageMilli = usage
			largestPod = podMetrics
		}
	}

	return largestPod, nil
}