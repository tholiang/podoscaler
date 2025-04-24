package main

import (
	"fmt"
	"math"
	"os"
	"time"

	util "github.com/tholiang/podoscaler/scalers/util"

	"k8s.io/client-go/kubernetes"
	kube_client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metrics_client "k8s.io/metrics/pkg/client/clientset/versioned"
)

/* --- GLOBAL VARS --- */
var clientset kube_client.Interface
var metrics_clientset *metrics_client.Clientset

const PROMETHEUS_URL = "http://prometheus.linkerd-viz.svc.cluster.local:9090"

const MIN_NODE_AVAILABILITY_THRESHOLD = 0.2
const DOWNSCALE_UTILIZATION_THRESHOLD = 0.85

// TODO: support multiple deployments
const DEPLOYMENT_NAME = "testapp"
const DEPLOYMENT_NAMESPACE = "default"
const MAPS = 500 // in millicpus
const LATENCY_THRESHOLD = 100 // in milliseconds

func main() {
	/* --- CONFIGURATION LOGIC --- */
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	// creates the clientset
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	metrics_clientset, err = metrics_client.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// set env variable for Prometheus service url
	os.Setenv("PROMETHEUS_URL", PROMETHEUS_URL)

	/* ------------ MAIN LOOP -------------- */
	for {
		fmt.Println("--- New Scaling Round ---")

		podList, err := util.GetPodListForDeployment(clientset, DEPLOYMENT_NAME, DEPLOYMENT_NAMESPACE)
		if err != nil {
			fmt.Printf("Failed to get pod list: %s\n", err.Error())
			continue
		}

		utilization, alloc, err := util.GetDeploymentUtilAndAlloc(clientset, metrics_clientset, DEPLOYMENT_NAME, DEPLOYMENT_NAMESPACE, podList)
		if err != nil {
			fmt.Printf("Failed to get average utilization and allocation: %s\n", err.Error())
			continue
		}
		utilPercent := float64(utilization) / float64(alloc)

		numPods := len(podList.Items)
		idealReplicaCt := int(math.Ceil(float64(utilization) / float64(MAPS)))
		newRequests := int64(math.Ceil(float64(utilization) / float64(idealReplicaCt)))

		if isSLOViolated() {
			// hscale
			if idealReplicaCt > numPods {
				hScale(idealReplicaCt)
				vScaleTo(newRequests)
				time.Sleep(10 * time.Second)
				continue
			}

			// vscale
			hasNoCongested := true
			for _, pod := range podList.Items {
				allocatable, capacity, err := util.GetNodeAllocableAndCapacity(clientset, pod.Spec.NodeName)
				if err != nil {
					fmt.Printf("Failed to get node allocatable and capacity: %s\n", err.Error())
					continue
				}
				
				availablePercentage := float64(allocatable) / float64(capacity)
				if availablePercentage > MIN_NODE_AVAILABILITY_THRESHOLD {
					continue
				}

				hasNoCongested = false
				currentRequests := pod.Spec.Containers[0].Resources.Requests.Cpu().MilliValue()
				additionalAllocation := newRequests - currentRequests
				if additionalAllocation > allocatable {
					// TODO: move pod greedily
				} else {
					vScaleTo(newRequests)
				}
			}

			if hasNoCongested {
				fmt.Println("External error detected, terminating autoscaler")
				return
			}

			time.Sleep(10 * time.Second)
		} else if utilPercent < DOWNSCALE_UTILIZATION_THRESHOLD {
			if idealReplicaCt < numPods {
				hScale(idealReplicaCt)
			}

			hysteresisMargin := 1 - DOWNSCALE_UTILIZATION_THRESHOLD
			newRequests = int64(math.Ceil(float64(newRequests) * hysteresisMargin))
			vScaleTo(newRequests)

			time.Sleep(10 * time.Second)
		}

		fmt.Println("--- Done ---")
		fmt.Println()
	}
}

func isSLOViolated() bool {
	prometheus_metrics, err := util.GetLatencyMetrics(DEPLOYMENT_NAME, 0.9)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}

	fmt.Printf("90th percentile latency: %f\n", prometheus_metrics[DEPLOYMENT_NAME])
	dist := prometheus_metrics[DEPLOYMENT_NAME] / LATENCY_THRESHOLD

	return dist > 1
}

// in-place scale all pods to the given CPU request
func vScaleTo(millis int64) error {
	podList, err := util.GetPodListForDeployment(clientset, DEPLOYMENT_NAME, DEPLOYMENT_NAMESPACE)
	if err != nil {
		return err
	}

	reqstr := fmt.Sprintf("%dm", millis)
	for _, pod := range podList.Items {
		container := pod.Spec.Containers[0] // TODO: handle multiple containers
		util.VScale(clientset, pod.Name, container.Name, reqstr)
	}

	return err
}

// TODO: make blocking/synchronous
func hScale(idealReplicaCt int) error {
	fmt.Printf("Changing count to %d\n", idealReplicaCt)

	return util.ChangeReplicaCount(DEPLOYMENT_NAMESPACE, DEPLOYMENT_NAME, idealReplicaCt, clientset)
}
