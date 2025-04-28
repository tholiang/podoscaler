package main

import (
	"fmt"
	"math"
	"os"
	"time"

	kube_client "k8s.io/client-go/kubernetes"
	metrics_client "k8s.io/metrics/pkg/client/clientset/versioned"
)

/* --- GLOBAL VARS --- */
const PROMETHEUS_URL = "http://prometheus.linkerd-viz.svc.cluster.local:9090"

const MIN_NODE_AVAILABILITY_THRESHOLD = 0.2
const DOWNSCALE_UTILIZATION_THRESHOLD = 0.85

const DEPLOYMENT_NAMESPACE = "default"
const MAPS = 500              // in millicpus
const LATENCY_THRESHOLD = 100 // in milliseconds

type Autoscaler struct {
	metrics           AutoscalerMetrics
	clientset         kube_client.Interface
	metrics_clientset *metrics_client.Clientset
}

func main() {
	am := new(DefaultAutoscalerMetrics)

	a := Autoscaler{}
	err := a.Init(am)
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

func (a *Autoscaler) Init(am AutoscalerMetrics) error {
	/* --- CONFIGURATION LOGIC --- */
	a.metrics = am

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
	os.Setenv("PROMETHEUS_URL", PROMETHEUS_URL)
	return nil
}

func (a *Autoscaler) RunRound() error {
	fmt.Println("--- New Scaling Round ---")

	// Get all deployments in the namespace
	deployments, err := a.metrics.GetAllDeploymentsFromNamespace(a.clientset, DEPLOYMENT_NAMESPACE)
	if err != nil {
		fmt.Printf("Failed to get deployments: %s\n", err.Error())
		return err
	}

	for _, deployment := range deployments.Items {
		deploymentName := deployment.Name
		fmt.Printf("Processing deployment: %s\n", deploymentName)

		podList, err := a.metrics.GetPodListForDeployment(a.clientset, deploymentName, DEPLOYMENT_NAMESPACE)
		if err != nil {
			fmt.Printf("Failed to get pod list for deployment %s: %s\n", deploymentName, err.Error())
			continue
		}

		utilization, alloc, err := a.metrics.GetDeploymentUtilAndAlloc(a.clientset, a.metrics_clientset, deploymentName, DEPLOYMENT_NAMESPACE, podList)
		if err != nil {
			fmt.Printf("Failed to get average utilization and allocation for deployment %s: %s\n", deploymentName, err.Error())
			continue
		}
		utilPercent := float64(utilization) / float64(alloc)
		fmt.Printf("Deployment %s: Utilization at %d of %d allocation\n", deploymentName, utilization, alloc)

		numPods := len(podList.Items)
		idealReplicaCt := int(math.Ceil(float64(utilization) / float64(MAPS)))
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
				usage, capacity, err := a.metrics.GetNodeUsageAndCapacity(a.clientset, a.metrics_clientset, pod.Spec.NodeName)
				if err != nil {
					fmt.Printf("Failed to get node usage and capacity for pod %s: %s\n", pod.Name, err.Error())
					continue
				}

				availableCPU := capacity - usage
				availablePercentage := float64(availableCPU) / float64(capacity)
				if availablePercentage > MIN_NODE_AVAILABILITY_THRESHOLD {
					continue
				}

				hasNoCongested = false
				currentRequests := pod.Spec.Containers[0].Resources.Requests.Cpu().MilliValue()
				additionalAllocation := newRequests - currentRequests
				fmt.Printf("%d %d\n", additionalAllocation, availableCPU)
				if additionalAllocation > availableCPU {
					// create new pod on uncongested node and delete old pod
					a.hScale(idealReplicaCt+1, deploymentName)
					a.metrics.DeletePod(a.clientset, pod.Name, DEPLOYMENT_NAMESPACE)
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
		} else if utilPercent < DOWNSCALE_UTILIZATION_THRESHOLD {
			if idealReplicaCt < numPods {
				a.hScale(idealReplicaCt, deploymentName)
			}

			hysteresisMargin := 1 / DOWNSCALE_UTILIZATION_THRESHOLD
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
	dist := prometheus_metrics[deploymentName] / LATENCY_THRESHOLD

	return dist > 1
}

// in-place scale all pods to the given CPU request
func (a *Autoscaler) vScaleTo(millis int64, deploymentName string) error {
	podList, err := a.metrics.GetPodListForDeployment(a.clientset, deploymentName, DEPLOYMENT_NAMESPACE)
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

	return a.metrics.ChangeReplicaCount(DEPLOYMENT_NAMESPACE, deploymentName, idealReplicaCt, a.clientset)
}
