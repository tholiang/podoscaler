package util

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	kube_client "k8s.io/client-go/kubernetes"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metrics_client "k8s.io/metrics/pkg/client/clientset/versioned"
)

func GetPodList(clientset kube_client.Interface, deploymentName, namespace string) (*v1.PodList, error) {
	ctx := context.TODO()

	// Get the Deployment
	deployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	// List Pods using the deployment's label selector
	labelSelector := labels.Set(deployment.Spec.Selector.MatchLabels).AsSelector().String()

	return clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
}

// in milliCPU
func GetPodSize(clientset kube_client.Interface, deploymentName, namespace string) (int64, error) {
	podList, err := GetPodList(clientset, deploymentName, namespace)
	if err != nil {
		return 0, err
	}

	if len(podList.Items) == 0 {
		return 0, fmt.Errorf("no pods found")
	}

	if len(podList.Items[0].Spec.Containers) == 0 {
		return 0, fmt.Errorf("no ready pods")
	}

	container := GetContainerByName(deploymentName, podList.Items[0].Spec.Containers)
	alloc := container.Resources.Requests["cpu"]
	return alloc.MilliValue(), nil
}

func GetPodMetrics(metricsClient metrics_client.Clientset, namespace string, podName string) (*v1beta1.PodMetrics, error) {
	return metricsClient.MetricsV1beta1().PodMetricses(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
}

func GetAllPodMetrics(clientset *metrics_client.Clientset) *v1beta1.PodMetricsList {
	podMetricsList, err := clientset.MetricsV1beta1().PodMetricses("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}

	return podMetricsList
}

// return average utilization of pods across a deployment as a percent of allocation
func GetAverageUtilization(clientset kube_client.Interface, metricsClient *metrics_client.Clientset, deploymentName, namespace string) (float64, error) {
	ctx := context.TODO()

	podList, err := GetPodList(clientset, deploymentName, namespace)
	if err != nil {
		return 0.0, fmt.Errorf("failed to get pods: %w", err)
	}

	sum := 0.0
	alive_pod_count := 0
	for _, pod := range podList.Items {
		container := GetContainerByName(deploymentName, pod.Spec.Containers)
		alloc := container.Resources.Requests["cpu"]
		allocval := alloc.MilliValue()

		podMetrics, err := metricsClient.MetricsV1beta1().PodMetricses(namespace).Get(ctx, pod.Name, metav1.GetOptions{})
		if err != nil {
			fmt.Printf("Warning: could not get metrics for pod %s: %v\n", pod.Name, err)
			continue
		}
		alive_pod_count += 1

		metric_container := GetMetricContainerByName(deploymentName, podMetrics.Containers)
		usage := metric_container.Usage.Cpu().MilliValue()
		fmt.Printf("pod %s\n- alloc: %dm\n- usage: %dm\n", pod.Name, allocval, usage)

		sum += float64(usage) / float64(allocval)
	}

	if alive_pod_count == 0 {
		return 0, nil
	}
	return sum / float64(alive_pod_count), nil
}

func GetSmallestPodOfDeployment(clientset kube_client.Interface, metricsClient *metrics_client.Clientset, deploymentName, namespace string) (*v1beta1.PodMetrics, error) {
	ctx := context.TODO()

	podList, err := GetPodList(clientset, deploymentName, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get pods: %w", err)
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
		
		metricContainer := GetMetricContainerByName(deploymentName,  podMetrics.Containers)
		usage := metricContainer.Usage.Cpu().MilliValue()
		if usage < smallestCPUUsageMilli {
			smallestCPUUsageMilli = usage
			smallestPod = podMetrics
		}
	}

	return smallestPod, nil
}

func GetLargestPodOfDeployment(clientset kube_client.Interface, metricsClient *metrics_client.Clientset, deploymentName, namespace string) (*v1beta1.PodMetrics, error) {
	ctx := context.TODO()

	podList, err := GetPodList(clientset, deploymentName, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get pods: %w", err)
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

		metricContainer := GetMetricContainerByName(deploymentName,  podMetrics.Containers)
		usage := metricContainer.Usage.Cpu().MilliValue()
		if usage > largestCPUUsageMilli {
			largestCPUUsageMilli = usage
			largestPod = podMetrics
		}
	}

	return largestPod, nil
}

func GetCongestedPods(clientset kube_client.Interface, metricsClient *metrics_client.Clientset, deploymentName, namespace string, congestionThreshold float64) ([]string, error) {
	podList, err := GetPodList(clientset, deploymentName, namespace)
	if err != nil {
		return []string{}, fmt.Errorf("failed to get pods: %w", err)
	}
	if len(podList.Items) == 0 {
		return []string{}, fmt.Errorf("no pods found for deployment %q", deploymentName)
	}

	// List all nodes
	congested_nodes, err := GetCongestedNodes(clientset, congestionThreshold)
	if err != nil {
		return []string{}, fmt.Errorf("failed to get congested nodes: %w", err)
	}

	congested_pods := []string{}
	for _, pod := range podList.Items {
		if slices.Contains(congested_nodes, pod.Spec.NodeName) {
			congested_pods = append(congested_pods, pod.Name)
		}
	}

	return congested_pods, nil
}

func GetCongestedNodes(clientset kube_client.Interface, congestionThreshold float64) ([]string, error) {
	nodeList, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return []string{}, err
	}

	congested_nodes := []string{}
	for _, node := range nodeList.Items {
		capacity := node.Status.Capacity.Cpu().MilliValue()
		allocatable := node.Status.Allocatable.Cpu().MilliValue()

		if float64(allocatable)/float64(capacity) < 1-congestionThreshold {
			congested_nodes = append(congested_nodes, node.Name)
		}
	}

	return congested_nodes, nil
}

func GetContainerByName(name string, containers []v1.Container) *v1.Container {
	for _, container := range containers {
		if strings.HasPrefix(container.Name, name) {
			return &container
		}
	}

	return nil
}

func GetMetricContainerByName(name string, containers []v1beta1.ContainerMetrics) *v1beta1.ContainerMetrics {
	for _, container := range containers {
		if strings.HasPrefix(container.Name, name) {
			return &container
		}
	}

	return nil
}