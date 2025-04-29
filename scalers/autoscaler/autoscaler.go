package main

import (
	"fmt"
	"math"
	"os"
	"time"

	kube_client "k8s.io/client-go/kubernetes"
	metrics_client "k8s.io/metrics/pkg/client/clientset/versioned"
)

/* --- CONFIG VARS --- */
const DEFAULT_PROMETHEUS_URL = "http://prometheus.linkerd-viz.svc.cluster.local:9090"

const DEFAULT_MIN_NODE_AVAILABILITY_THRESHOLD = 0.2
const DEFAULT_DOWNSCALE_UTILIZATION_THRESHOLD = 0.85

const DEFAULT_DEPLOYMENT_NAMESPACE = "default"
const DEFAULT_MAPS = 500              // in millicpus
const DEFAULT_LATENCY_THRESHOLD = 100 // in milliseconds

type Autoscaler struct {
	prometheus_url                   string
	min_node_availabiility_threshold float64
	downscale_utilization_threshold  float64
	deployment_namespace             string
	maps                             int64
	latency_threshold                int64

	metrics           AutoscalerMetrics
	clientset         kube_client.Interface
	metrics_clientset *metrics_client.Clientset
}

func main() {
	am := new(DefaultAutoscalerMetrics)

	a := Autoscaler{
		prometheus_url:                   DEFAULT_PROMETHEUS_URL,
		min_node_availabiility_threshold: DEFAULT_MIN_NODE_AVAILABILITY_THRESHOLD,
		downscale_utilization_threshold:  DEFAULT_DOWNSCALE_UTILIZATION_THRESHOLD,
		deployment_namespace:             DEFAULT_DEPLOYMENT_NAMESPACE,
		maps:                             DEFAULT_MAPS,
		latency_threshold:                DEFAULT_LATENCY_THRESHOLD,
		metrics:                          am,
	}
	err := a.Init()
	if err != nil {
		panic(err)
	}

	for {
		err := a.RunRound()
		if err != nil {
			panic(err)
		}

		time.Sleep(5 * time.Second)
	}
}

func (a *Autoscaler) Init() error {
	/* --- CONFIGURATION LOGIC --- */
	// creates the in-cluster config
	config, err := a.metrics.GetKubernetesConfig()
	if err != nil {
		return err
	}

	// creates the clientset
	a.clientset, err = a.metrics.GetClientset(config)
	if err != nil {
		return err
	}
	a.metrics_clientset, err = a.metrics.GetMetricsClientset(config)
	if err != nil {
		return err
	}

	// set env variable for Prometheus service url
	os.Setenv("PROMETHEUS_URL", a.prometheus_url)
	return nil
}

func (a *Autoscaler) RunRound() error {
	fmt.Println("--- New Scaling Round ---")

	// Get all deployments in the namespace
	deployments, err := a.metrics.GetAllDeploymentsFromNamespace(a.clientset, a.deployment_namespace)
	if err != nil {
		fmt.Printf("Failed to get deployments: %s\n", err.Error())
		return err
	}

	for _, deployment := range deployments.Items {
		deploymentName := deployment.Name
		fmt.Printf("Processing deployment: %s\n", deploymentName)

		podList, err := a.metrics.GetPodListForDeployment(a.clientset, deploymentName, a.deployment_namespace)
		if err != nil {
			fmt.Printf("Failed to get pod list for deployment %s: %s\n", deploymentName, err.Error())
			continue
		}

		utilization, alloc, err := a.metrics.GetDeploymentUtilAndAlloc(a.clientset, a.metrics_clientset, deploymentName, a.deployment_namespace, podList)
		if err != nil {
			fmt.Printf("Failed to get average utilization and allocation for deployment %s: %s\n", deploymentName, err.Error())
			continue
		}
		utilPercent := float64(utilization) / float64(alloc)
		fmt.Printf("Deployment %s: Utilization at %d of %d allocation\n", deploymentName, utilization, alloc)

		numPods := len(podList.Items)
		idealReplicaCt := int(math.Ceil(float64(utilization) / float64(a.maps)))
		newRequests := int64(math.Ceil(float64(utilization) / float64(idealReplicaCt)))

		if a.isSLOViolated(deploymentName) {
			fmt.Printf("Deployment %s: Above SLO\n", deploymentName)
			// hscale
			if idealReplicaCt > numPods {
				a.hScale(idealReplicaCt, deploymentName)
				a.vScaleTo(newRequests, deploymentName)
				continue
			}

			// vscale
			hasNoCongested := true
			for _, pod := range podList.Items {
				usage, err := a.metrics.GetNodeUsage(a.metrics_clientset, pod.Spec.NodeName)
				if err != nil {
					fmt.Printf("Failed to get node usage for pod %s: %s\n", pod.Name, err.Error())
					continue
				}

				alloc, capacity, err := a.metrics.GetNodeAllocableAndCapacity(a.clientset, pod.Spec.NodeName)
				if err != nil {
					fmt.Printf("Failed to get node allocable and capacity for pod %s: %s\n", pod.Name, err.Error())
					continue
				}

				unusedCPU := capacity - usage
				unusedPercentage := float64(unusedCPU) / float64(capacity)
				if unusedPercentage > a.min_node_availabiility_threshold {
					continue
				}
				hasNoCongested = false

				currentRequests := pod.Spec.Containers[0].Resources.Requests.Cpu().MilliValue()
				unallocatedCPU := capacity - alloc
				additionalAllocation := newRequests - currentRequests
				if additionalAllocation > unallocatedCPU {
					// create new pod on uncongested node and delete old pod
					a.hScale(idealReplicaCt+1, deploymentName)
					a.metrics.DeletePod(a.clientset, pod.Name, a.deployment_namespace)
					a.hScale(idealReplicaCt, deploymentName)
				}
			}

			if hasNoCongested {
				fmt.Printf("Deployment %s: External error detected, exiting\n", deploymentName)
				return nil
			} else {
				a.vScaleTo(newRequests, deploymentName)
			}
			time.Sleep(5 * time.Second)
		} else if utilPercent < a.downscale_utilization_threshold {
			if idealReplicaCt < numPods {
				a.hScale(idealReplicaCt, deploymentName)
			}

			hysteresisMargin := 1 / a.downscale_utilization_threshold
			newRequests = int64(math.Ceil(float64(newRequests) * hysteresisMargin))
			a.vScaleTo(newRequests, deploymentName)
			time.Sleep(5 * time.Second)
		}
	}

	fmt.Println("--- Done ---")
	fmt.Println()

	return nil
}

func (a *Autoscaler) isSLOViolated(deploymentName string) bool {
	prometheus_metrics, err := a.metrics.GetLatencyMetrics(deploymentName, 0.9)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}

	fmt.Printf("90th percentile latency: %f\n", prometheus_metrics[deploymentName])
	dist := prometheus_metrics[deploymentName] / float64(a.latency_threshold)

	return dist > 1
}

// in-place scale all pods to the given CPU request
func (a *Autoscaler) vScaleTo(millis int64, deploymentName string) error {
	podList, err := a.metrics.GetPodListForDeployment(a.clientset, deploymentName, a.deployment_namespace)
	if err != nil {
		return err
	}

	reqstr := fmt.Sprintf("%dm", millis)
	for _, pod := range podList.Items {
		container := pod.Spec.Containers[0] // TODO: handle multiple containers
		err = a.metrics.VScale(a.clientset, pod.Name, container.Name, reqstr)
	}

	return err
}

// is blocking (see `hScaleFromHSR`)
func (a *Autoscaler) hScale(idealReplicaCt int, deploymentName string) error {
	fmt.Printf("Changing count to %d\n", idealReplicaCt)

	return a.metrics.ChangeReplicaCount(a.deployment_namespace, deploymentName, idealReplicaCt, a.clientset)
}
