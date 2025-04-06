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

const PROMETHEUS_URL = "http://10.107.52.199:9090" // change to ip of svc "prometheus-kube-prometheus-prometheus"

const LATENCY_SLO = 10 // ms
const SLO_LOWER_THRESHOLD = 0.85
const DEPLOYMENT_NAME = "testapp"
const DEPLOYMENT_NAMESPACE = "default"
const SERVICE_NAME = "testapp-service"
const SERVICE_ENDPOINT = "/noop"

const NODE_CONGESTION_THRESHOLD = 0.9
const DOWNSCALE_UTILIZATION_THRESHOLD = 0.85

const SCALE_UP_MULTIPLIER = 1.2
const SCALE_DOWN_MULTIPLIER = 0.8
const MAX_APS = 300 // profiled per deployment
const MIN_APS = 100 // CPU millivalue

func main() {
	/* --- K8S CLIENT GO CONFIG STUFF --- */
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

	for {
		time.Sleep(5 * time.Second)

		fmt.Println("---New Scaling Round---")

		status := getSLOStatus()
		utilization, err := util.GetAverageUtilization(clientset, metrics_clientset, DEPLOYMENT_NAME, DEPLOYMENT_NAMESPACE)
		if err != nil {
			fmt.Printf("failed to get average utilization: %s\n", err.Error())
			continue
		}

		if status > 0 {
			fmt.Printf("Over SLO, CPU utilization at %f\n", utilization)

			if utilization > 1.0 {
				fmt.Println("---Upscaling!!---")

				congested_pods, err := util.GetCongestedPods(clientset, metrics_clientset, DEPLOYMENT_NAME, DEPLOYMENT_NAMESPACE, NODE_CONGESTION_THRESHOLD)
				if err != nil {
					fmt.Printf("failed to get average congested pod list: %s\n", err.Error())
					continue
				}

				if len(congested_pods) > 0 {
					fmt.Printf("Attempting to move %d pods to uncongested nodes\n", len(congested_pods))
					err = util.MovePods(clientset, DEPLOYMENT_NAMESPACE, DEPLOYMENT_NAME, congested_pods)
					if err != nil {
						fmt.Printf("failed to move pods: %s\n", err.Error())
						continue
					}
				}

				// now we can assume no pods are congested
				podSize, err := util.GetPodSize(clientset, DEPLOYMENT_NAME, DEPLOYMENT_NAMESPACE)
				if err != nil {
					fmt.Printf("failed to get pod size: %s\n", err.Error())
					continue
				}
				numReplicas, err := util.GetReplicaCt(clientset, DEPLOYMENT_NAME, DEPLOYMENT_NAMESPACE)
				if err != nil {
					fmt.Printf("failed to get number of replicas: %s\n", err.Error())
					continue
				}

				totalMilliUsage := int64(float64(podSize*int64(numReplicas)) * utilization)

				// hscale if needed
				if podSize == MAX_APS {
					idealReplicaCt := int(math.Ceil(float64(totalMilliUsage) / float64(MAX_APS)))
					fmt.Printf("hscaling to %d replicas\n", idealReplicaCt)
					err := hScale(idealReplicaCt - numReplicas)
					if err != nil {
						fmt.Printf("hscale failed: %s\n", err.Error())
						continue
					}
					numReplicas = idealReplicaCt
				}

				// finally vscale to ideal size
				newPodSize := min(MAX_APS, totalMilliUsage/int64(numReplicas))
				fmt.Printf("vscaling to %dm CPU\n", newPodSize)
				err = vScaleTo(newPodSize)
				if err != nil {
					fmt.Printf("vscale failed: %s\n", err.Error())
					continue
				}
			}
		} else if status < 0 {
			fmt.Printf(" Below SLO, CPU utilization at %f\n", utilization)

			if utilization < DOWNSCALE_UTILIZATION_THRESHOLD {
				fmt.Println("---Downscaling!!---")

				podSize, err := util.GetPodSize(clientset, DEPLOYMENT_NAME, DEPLOYMENT_NAMESPACE)
				if err != nil {
					fmt.Printf("failed to get pod size: %s\n", err.Error())
					continue
				}
				numReplicas, err := util.GetReplicaCt(clientset, DEPLOYMENT_NAME, DEPLOYMENT_NAMESPACE)
				if err != nil {
					fmt.Printf("failed to get number of replicas: %s\n", err.Error())
					continue
				}

				totalMilliUsage := int64(float64(podSize*int64(numReplicas)) * utilization)

				// hscale if needed
				// TODO: may need to check if theres space on available nodes for vscale after
				idealReplicaCt := int(math.Ceil(float64(totalMilliUsage) / float64(MAX_APS)))
				if idealReplicaCt < numReplicas {
					fmt.Printf("HScaling to %d replicas\n", idealReplicaCt)
					err := hScale(idealReplicaCt - numReplicas)
					if err != nil {
						fmt.Printf("hscale failed: %s\n", err.Error())
						continue
					}
					numReplicas = idealReplicaCt
				}

				// finally vscale to ideal size
				newPodSize := min(MAX_APS, totalMilliUsage/int64(numReplicas))
				fmt.Printf("VScaling to %dm CPU\n", newPodSize)
				err = vScaleTo(newPodSize)
				if err != nil {
					fmt.Printf("vscale failed: %s\n", err.Error())
					continue
				}
			}
		}

		fmt.Println("---Done---")
		fmt.Println()
	}
}

// return 1 if above SLO, 0 if at SLO, and -1 if below SLO
func getSLOStatus() int {
	prometheus_metrics, err := util.GetLatencyMetrics(SERVICE_NAME, 0.9)
	if err != nil {
		fmt.Println(err.Error())
		return 0
	}

	dist = prometheus_metrics[SERVICE_ENDPOINT] / LATENCY_SLO
	
	if dist > 1 {
		return 1
	} else if dist < SLO_LOWER_THRESHOLD {
		return -1
	}

	return 0
}

// in-place scale all pods to
func vScaleTo(millis int64) error {
	podList, err := util.GetPodList(clientset, DEPLOYMENT_NAME, DEPLOYMENT_NAMESPACE)
	if err != nil {
		return err
	}

	reqstr := fmt.Sprintf("%dm", millis)
	for _, pod := range podList.Items {
		util.VScale(clientset, pod.Name, pod.Spec.Containers[0].Name, reqstr, "1000000m")
	}

	return err
}

// in-place scale up all pods in deployment
func vScaleUp() error {
	podList, err := util.GetPodList(clientset, DEPLOYMENT_NAME, DEPLOYMENT_NAMESPACE)
	if err != nil {
		return err
	}

	for _, pod := range podList.Items {
		podMetrics, meterr := util.GetPodMetrics(*metrics_clientset, DEPLOYMENT_NAMESPACE, pod.Name)
		if meterr != nil {
			err = meterr
			continue
		}
		util.VScalePod(podMetrics, SCALE_UP_MULTIPLIER, clientset)
	}

	return err
}

// in-place scale down all pods in deployment
func vScaleDown() error {
	podList, err := util.GetPodList(clientset, DEPLOYMENT_NAME, DEPLOYMENT_NAMESPACE)
	if err != nil {
		return err
	}

	for _, pod := range podList.Items {
		podMetrics, meterr := util.GetPodMetrics(*metrics_clientset, DEPLOYMENT_NAMESPACE, pod.Name)
		if meterr != nil {
			err = meterr
			continue
		}
		util.VScalePod(podMetrics, SCALE_DOWN_MULTIPLIER, clientset)
	}

	return err
}

// add or remove `delta` amount of replicas in deployment
func hScale(delta int) error {
	replicaCt, err := util.GetReplicaCt(clientset, DEPLOYMENT_NAME, DEPLOYMENT_NAMESPACE)
	if err != nil {
		return err
	}

	fmt.Printf("Current replica count %d. Changing count to %d\n", replicaCt, replicaCt+delta)
	return util.ChangeReplicaCount(DEPLOYMENT_NAMESPACE, DEPLOYMENT_NAME, replicaCt+delta, clientset)
}
