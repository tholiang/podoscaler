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

// TODO: support multiple deployments
const DEPLOYMENT_NAME = "testapp"
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

		time.Sleep(10 * time.Second)
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
	/* ------------ MAIN LOOP -------------- */
	fmt.Println("--- New Scaling Round ---")

	podList, err := a.metrics.GetPodListForDeployment(a.clientset, DEPLOYMENT_NAME, DEPLOYMENT_NAMESPACE)
	if err != nil {
		fmt.Printf("Failed to get pod list: %s\n", err.Error())
		return err
	}

	utilization, alloc, err := a.metrics.GetDeploymentUtilAndAlloc(a.clientset, a.metrics_clientset, DEPLOYMENT_NAME, DEPLOYMENT_NAMESPACE, podList)
	if err != nil {
		fmt.Printf("Failed to get average utilization and allocation: %s\n", err.Error())
		return err
	}
	utilPercent := float64(utilization) / float64(alloc)
	fmt.Printf("Utilization at %d of %d allocation\n", utilization, alloc)

	numPods := len(podList.Items)
	idealReplicaCt := int(math.Ceil(float64(utilization) / float64(MAPS)))
	newRequests := int64(math.Ceil(float64(utilization) / float64(idealReplicaCt)))

	if a.isSLOViolated() {
		fmt.Println("Above SLO")
		// hscale
		if idealReplicaCt > numPods {
			a.hScale(idealReplicaCt)
			a.vScaleTo(newRequests)
			return nil
		}

		// vscale
		hasNoCongested := true
		for _, pod := range podList.Items {
			allocatable, capacity, err := a.metrics.GetNodeAllocableAndCapacity(a.clientset, pod.Spec.NodeName)
			if err != nil {
				fmt.Printf("Failed to get node allocatable and capacity: %s\n", err.Error())
				return err
			}

			availablePercentage := float64(allocatable) / float64(capacity)
			if availablePercentage > MIN_NODE_AVAILABILITY_THRESHOLD {
				return nil
			}

			hasNoCongested = false
			currentRequests := pod.Spec.Containers[0].Resources.Requests.Cpu().MilliValue()
			additionalAllocation := newRequests - currentRequests
			fmt.Printf("%d %d\n", additionalAllocation, allocatable)
			if additionalAllocation > allocatable {
				// TODO: move pod greedily
			} else {
				a.vScaleTo(newRequests)
			}
		}

		if hasNoCongested {
			fmt.Println("External error detected, terminating autoscaler")
			return nil
		}
	} else if utilPercent < DOWNSCALE_UTILIZATION_THRESHOLD {
		if idealReplicaCt < numPods {
			a.hScale(idealReplicaCt)
		}

		hysteresisMargin := 1 / DOWNSCALE_UTILIZATION_THRESHOLD
		newRequests = int64(math.Ceil(float64(newRequests) * hysteresisMargin))
		a.vScaleTo(newRequests)
	}

	fmt.Println("--- Done ---")
	fmt.Println()

	return nil
}

func (a *Autoscaler) isSLOViolated() bool {
	prometheus_metrics, err := a.metrics.GetLatencyMetrics(DEPLOYMENT_NAME, 0.9)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}

	fmt.Printf("90th percentile latency: %f\n", prometheus_metrics[DEPLOYMENT_NAME])
	dist := prometheus_metrics[DEPLOYMENT_NAME] / LATENCY_THRESHOLD

	return dist > 1
}

// in-place scale all pods to the given CPU request
func (a *Autoscaler) vScaleTo(millis int64) error {
	podList, err := a.metrics.GetPodListForDeployment(a.clientset, DEPLOYMENT_NAME, DEPLOYMENT_NAMESPACE)
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

// TODO: make blocking/synchronous
func (a *Autoscaler) hScale(idealReplicaCt int) error {
	fmt.Printf("Changing count to %d\n", idealReplicaCt)

	return a.metrics.ChangeReplicaCount(DEPLOYMENT_NAMESPACE, DEPLOYMENT_NAME, idealReplicaCt, a.clientset)
}
