//go:build autoscaler || autoscalertest
// +build autoscaler autoscalertest

package autoscaler

import (
	"fmt"
	"math"
	"os"
	"time"

	kube_client "k8s.io/client-go/kubernetes"
	metrics_client "k8s.io/metrics/pkg/client/clientset/versioned"
)

/* --- CONFIG VARS --- */
const (
	DEFAULT_MIN_NODE_AVAILABILITY_THRESHOLD = 0.4
	DEFAULT_DOWNSCALE_UTILIZATION_THRESHOLD = 0.85
)

const (
	DEFAULT_MAPS              = 500 // in millicpus
	DEFAULT_MIN_REQUESTS      = 100 // in millicpus
	DEFAULT_LATENCY_THRESHOLD = 40  // in milliseconds
)

type Autoscaler struct {
	PrometheusUrl                 string
	MinNodeAvailabilityThreshold  float64
	DownscaleUtilizationThreshold float64
	Maps                          int64
	LatencyThreshold              int64

	Metrics          AutoscalerMetrics
	Clientset        kube_client.Interface
	MetricsClientset *metrics_client.Clientset
}

func (a *Autoscaler) Init() error {
	/* --- CONFIGURATION LOGIC --- */
	// creates the in-cluster config
	config, err := a.Metrics.GetKubernetesConfig()
	if err != nil {
		return err
	}

	// creates the clientset
	a.Clientset, err = a.Metrics.GetClientset(config)
	if err != nil {
		return err
	}
	a.MetricsClientset, err = a.Metrics.GetMetricsClientset(config)
	if err != nil {
		return err
	}

	// set env variable for Prometheus service url
	os.Setenv("PROMETHEUS_URL", a.PrometheusUrl)
	return nil
}

func (a *Autoscaler) RunRound() error {
	fmt.Printf("\n=== Autoscaler Round %s ===\n", time.Now().Format(time.RFC3339))

	// get node usages
	fmt.Printf("\nGetting node usages...\n")
	nodelist, err := a.Metrics.GetNodeList(a.Clientset)
	if err != nil {
		fmt.Printf("❌ ERROR: Failed to get node list: %s\n", err.Error())
		return err
	}

	for _, node := range nodelist.Items {
		nodeName := node.Name
		usage, err := a.Metrics.GetNodeUsage(a.MetricsClientset, nodeName)
		if err != nil {
			fmt.Printf("❌ ERROR: Failed to get usage for node %s: %s\n", nodeName, err.Error())
			continue
		}

		allocable, capacity, err := a.Metrics.GetNodeAllocableAndCapacity(a.Clientset, nodeName)
		if err != nil {
			fmt.Printf("❌ ERROR: Failed to get node metrics for node %s: %s\n", nodeName, err.Error())
			continue
		}

		fmt.Printf("%s: %d in use, %d allocable, %d capacity\n", nodeName, usage, allocable, capacity)
	}

	// Get all deployments in the namespace
	deployments, err := a.Metrics.GetControlledDeployments(a.Clientset)
	if err != nil {
		fmt.Printf("❌ ERROR: Failed to get deployments: %s\n", err.Error())
		return err
	}

	for _, deployment := range deployments.Items {
		deploymentName := deployment.Name
		deploymentNamespace := deployment.Namespace
		fmt.Printf("\n📦 Processing deployment: %s\n", deploymentName)

		podList, err := a.Metrics.GetReadyPodListForDeployment(a.Clientset, deploymentName, deploymentNamespace)
		if err != nil {
			fmt.Printf("❌ ERROR: Failed to get pod list for deployment %s: %s\n", deploymentName, err.Error())
			continue
		}

		utilization, alloc, err := a.Metrics.GetDeploymentUtilAndAlloc(a.Clientset, a.MetricsClientset, deploymentName, deploymentNamespace, podList)
		if err != nil {
			fmt.Printf("❌ ERROR: Failed to get utilization metrics for deployment %s: %s\n", deploymentName, err.Error())
			continue
		}
		utilPercent := float64(utilization) / float64(alloc)
		fmt.Printf("📊 Current state: %d/%d millicpus (%.1f%%)\n", utilization, alloc, utilPercent*100)

		numPods := len(podList)
		idealReplicaCt := int(math.Ceil(float64(utilization) / float64(a.Maps)))
		newRequests := int64(math.Ceil(float64(utilization) / float64(idealReplicaCt)))

		perpodalloc := int64(math.Ceil(float64(alloc) / float64(numPods)))

		if a.isSLOViolated(deploymentName) {
			fmt.Printf("⚠️ SLO violation detected for %s\n", deploymentName)
			// hscale
			if idealReplicaCt > numPods {
				fmt.Printf("🔄 Horizontal scaling: %d -> %d replicas\n", numPods, idealReplicaCt)
				err = a.hScale(idealReplicaCt, deploymentName, deploymentNamespace)
				if err != nil {
					fmt.Printf("❌ ERROR: Failed to hscale deployment %s: %s\n", deploymentName, err.Error())
					continue
				}
				fmt.Printf("🔄 Vertical scaling: %d -> %d millicpus\n", perpodalloc, newRequests)
				err = a.vScaleTo(newRequests, deploymentName, deploymentNamespace)
				if err != nil {
					fmt.Printf("❌ ERROR: Failed to vscale deployment %s: %s\n", deploymentName, err.Error())
					continue
				}
				continue
			}

			// vscale
			hasNoCongested := true
			for _, pod := range podList {
				usage, err := a.Metrics.GetNodeUsage(a.MetricsClientset, pod.Spec.NodeName)
				if err != nil {
					fmt.Printf("❌ ERROR: Failed to get node usage for pod %s: %s\n", pod.Name, err.Error())
					continue
				}

				allocable, capacity, err := a.Metrics.GetNodeAllocableAndCapacity(a.Clientset, pod.Spec.NodeName)
				if err != nil {
					fmt.Printf("❌ ERROR: Failed to get node metrics for pod %s: %s\n", pod.Name, err.Error())
					continue
				}

				unusedCPU := min(capacity-usage, allocable)
				unusedPercentage := float64(unusedCPU) / float64(capacity)
				if unusedPercentage > a.MinNodeAvailabilityThreshold {
					continue
				}
				hasNoCongested = false

				idx := 0
				if pod.Spec.Containers[0].Name == "linkerd-proxy" {
					idx = 1
				}
				currentRequests := pod.Spec.Containers[idx].Resources.Requests.Cpu().MilliValue()
				additionalAllocation := newRequests - currentRequests
				if additionalAllocation > allocable {
					fmt.Printf("🔄 Node migration: Moving pod %s to uncongested node\n", pod.Name)
					err = a.hScale(idealReplicaCt+1, deploymentName, deploymentNamespace)
					if err != nil {
						fmt.Printf("❌ ERROR: Failed to add affinity pod for deployment %s: %s\n", deploymentName, err.Error())
						continue
					}
					a.Metrics.DeletePod(a.Clientset, pod.Name, deploymentNamespace)
					err = a.hScale(idealReplicaCt, deploymentName, deploymentNamespace)
					if err != nil {
						fmt.Printf("❌ ERROR: Failed to remove affinity pod for deployment %s: %s\n", deploymentName, err.Error())
						continue
					}
				}
			}

			if hasNoCongested || newRequests < perpodalloc {
				fmt.Printf("ℹ️ External bottleneck detected for %s - no action taken\n", deploymentName)
				continue
			} else {
				fmt.Printf("🔄 Vertical scaling: %d -> %d millicpus\n", perpodalloc, newRequests)
				err = a.vScaleTo(newRequests, deploymentName, deploymentNamespace)
				if err != nil {
					fmt.Printf("❌ ERROR: Failed to vscale deployment %s: %s\n", deploymentName, err.Error())
					continue
				}
			}
		} else if utilPercent < a.DownscaleUtilizationThreshold {
			idealReplicaCt = max(idealReplicaCt, 1)
			if idealReplicaCt < numPods {
				fmt.Printf("🔄 Downscaling: %d -> %d replicas\n", numPods, idealReplicaCt)
				err = a.hScale(max(idealReplicaCt, 1), deploymentName, deploymentNamespace)
				if err != nil {
					fmt.Printf("❌ ERROR: Failed to hscale down deployment %s: %s\n", deploymentName, err.Error())
					continue
				}
			}

			hysteresisMargin := 1 / a.DownscaleUtilizationThreshold
			newRequests = int64(math.Ceil(float64(newRequests) * hysteresisMargin))
			newRequests = max(newRequests, DEFAULT_MIN_REQUESTS)
			if newRequests == perpodalloc {
				continue
			}

			fmt.Printf("🔄 Downscaling: %d -> %d millicpus\n", perpodalloc, newRequests)
			err = a.vScaleTo(newRequests, deploymentName, deploymentNamespace)
			if err != nil {
				fmt.Printf("❌ ERROR: Failed to vscale down deployment %s: %s\n", deploymentName, err.Error())
				continue
			}
		}
	}

	fmt.Printf("\n=== Round completed ===\n\n")
	return nil
}

func (a *Autoscaler) isSLOViolated(deploymentName string) bool {
	metrics, err := a.Metrics.GetLatencyMetrics(a.Clientset)
	if err != nil {
		fmt.Printf("❌ ERROR: Failed to get latency metrics for %s: %s\n", deploymentName, err.Error())
		return false
	}

	if len(metrics) == 0 {
		fmt.Printf("ℹ️ No latency metrics found for deployment %s\n", deploymentName)
		return false
	}
	latency := metrics["p99"] * 1000 // convert from s to ms
	dist := latency / float64(a.LatencyThreshold)
	fmt.Printf("📊 Latency metrics: %.2fms (threshold: %dms)\n", latency, a.LatencyThreshold)

	return dist > 1
}

// in-place scale all pods to the given CPU request
func (a *Autoscaler) vScaleTo(millis int64, deploymentName string, deploymentNamespace string) error {
	podList, err := a.Metrics.GetReadyPodListForDeployment(a.Clientset, deploymentName, deploymentNamespace)
	if err != nil {
		return err
	}

	reqstr := fmt.Sprintf("%dm", millis)
	for _, pod := range podList {
		idx := 0
		if pod.Spec.Containers[0].Name == "linkerd-proxy" {
			idx = 1
		}
		container := pod.Spec.Containers[idx] // TODO: handle multiple containers
		err = a.Metrics.VScale(a.Clientset, pod.Name, container.Name, reqstr, deploymentNamespace)
		if err != nil {
			fmt.Printf("Failed to vscale pod %s: %s\n", pod.Name, err.Error())
			return err
		}
	}

	return nil
}

// is blocking (see `hScaleFromHSR`)
func (a *Autoscaler) hScale(idealReplicaCt int, deploymentName string, deploymentNamespace string) error {
	return a.Metrics.ChangeReplicaCount(deploymentNamespace, deploymentName, idealReplicaCt, a.Clientset)
}
