package autoscaler

import (
	"fmt"
	"math"
	"os"

	kube_client "k8s.io/client-go/kubernetes"
	metrics_client "k8s.io/metrics/pkg/client/clientset/versioned"
)

/* --- CONFIG VARS --- */
const DEFAULT_PROMETHEUS_URL = "http://prometheus.linkerd-viz.svc.cluster.local:9090"

const DEFAULT_MIN_NODE_AVAILABILITY_THRESHOLD = 0.2
const DEFAULT_DOWNSCALE_UTILIZATION_THRESHOLD = 0.85

const DEFAULT_MAPS = 500              // in millicpus
const DEFAULT_LATENCY_THRESHOLD = 40 // in milliseconds

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
	fmt.Println("--- New Scaling Round ---")

	// Get all deployments in the namespace
	deployments, err := a.Metrics.GetControlledDeployments(a.Clientset)
	if err != nil {
		fmt.Printf("Failed to get deployments: %s\n", err.Error())
		return err
	}

	for _, deployment := range deployments.Items {
		deploymentName := deployment.Name
		deploymentNamespace := deployment.Namespace
		fmt.Printf("Processing deployment: %s\n", deploymentName)

		podList, err := a.Metrics.GetReadyPodListForDeployment(a.Clientset, deploymentName, deploymentNamespace)
		if err != nil {
			fmt.Printf("Failed to get pod list for deployment %s: %s\n", deploymentName, err.Error())
			continue
		}

		utilization, alloc, err := a.Metrics.GetDeploymentUtilAndAlloc(a.Clientset, a.MetricsClientset, deploymentName, deploymentNamespace, podList)
		if err != nil {
			fmt.Printf("Failed to get average utilization and allocation for deployment %s: %s\n", deploymentName, err.Error())
			continue
		}
		utilPercent := float64(utilization) / float64(alloc)
		fmt.Printf("Deployment %s: Utilization at %d of %d allocation\n", deploymentName, utilization, alloc)

		numPods := len(podList)
		idealReplicaCt := int(math.Ceil(float64(utilization) / float64(a.Maps)))
		newRequests := int64(math.Ceil(float64(utilization) / float64(idealReplicaCt)))

		if a.isSLOViolated(deploymentName) {
			fmt.Printf("Deployment %s: Above SLO\n", deploymentName)
			// hscale
			if idealReplicaCt > numPods {
				err = a.hScale(idealReplicaCt, deploymentName, deploymentNamespace)
				if err != nil {
					fmt.Printf("HSCALE: failed to hscale deployment %s: %s\n", deploymentName, err.Error())
					continue
				}
				err = a.vScaleTo(newRequests, deploymentName, deploymentNamespace)
				if err != nil {
					fmt.Printf("HSCALE: failed to vscale deployment %s: %s\n", deploymentName, err.Error())
					continue
				}
				continue
			}

			// vscale
			hasNoCongested := true
			for _, pod := range podList {
				usage, err := a.Metrics.GetNodeUsage(a.MetricsClientset, pod.Spec.NodeName)
				if err != nil {
					fmt.Printf("Failed to get node usage for pod %s: %s\n", pod.Name, err.Error())
					continue
				}

				allocable, capacity, err := a.Metrics.GetNodeAllocableAndCapacity(a.Clientset, pod.Spec.NodeName)
				if err != nil {
					fmt.Printf("Failed to get node allocable and capacity for pod %s: %s\n", pod.Name, err.Error())
					continue
				}

				unusedCPU := capacity - usage
				unusedPercentage := float64(unusedCPU) / float64(capacity)
				if unusedPercentage > a.MinNodeAvailabilityThreshold {
					continue
				}
				hasNoCongested = false

				currentRequests := pod.Spec.Containers[0].Resources.Requests.Cpu().MilliValue()
				additionalAllocation := newRequests - currentRequests
				if additionalAllocation > allocable {
					// create new pod on uncongested node and delete old pod
					err = a.hScale(idealReplicaCt+1, deploymentName, deploymentNamespace)
					if err != nil {
						fmt.Printf("Failed to add afinity pod for deployment %s: %s\n", deploymentName, err.Error())
						continue
					}
					a.Metrics.DeletePod(a.Clientset, pod.Name, deploymentNamespace)
					err = a.hScale(idealReplicaCt, deploymentName, deploymentNamespace)
					if err != nil {
						fmt.Printf("Failed to remove afinity pod for deployment %s: %s\n", deploymentName, err.Error())
						continue
					}
				}
			}

			if hasNoCongested {
				fmt.Printf("Deployment %s: External bottleneck detected; doing nothing\n", deploymentName)
				return nil
			} else {
				err = a.vScaleTo(newRequests, deploymentName, deploymentNamespace)
				if err != nil {
					fmt.Printf("VSCALE: failed to vscale deployment %s: %s\n", deploymentName, err.Error())
					continue
				}
			}
		} else if utilPercent < a.DownscaleUtilizationThreshold {
			if idealReplicaCt < numPods {
				err = a.hScale(idealReplicaCt, deploymentName, deploymentNamespace)
				if err != nil {
					fmt.Printf("Failed to hscale down deployment %s: %s\n", deploymentName, err.Error())
					continue
				}
			}

			hysteresisMargin := 1 / a.DownscaleUtilizationThreshold
			newRequests = int64(math.Ceil(float64(newRequests) * hysteresisMargin))
			err = a.vScaleTo(newRequests, deploymentName, deploymentNamespace)
			if err != nil {
				fmt.Printf("Failed to vscale down deployment %s: %s\n", deploymentName, err.Error())
				continue
			}
		}
	}

	fmt.Println("--- Done ---")
	fmt.Println()

	return nil
}

func (a *Autoscaler) isSLOViolated(deploymentName string) bool {
	prometheus_metrics, err := a.Metrics.GetLatencyMetrics(deploymentName, 0.9)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}

	fmt.Printf("90th percentile latency: %f\n", prometheus_metrics[deploymentName])
	dist := prometheus_metrics[deploymentName] / float64(a.LatencyThreshold)

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
		container := pod.Spec.Containers[0] // TODO: handle multiple containers
		err = a.Metrics.VScale(a.Clientset, pod.Name, container.Name, reqstr)
	}

	return err
}

// is blocking (see `hScaleFromHSR`)
func (a *Autoscaler) hScale(idealReplicaCt int, deploymentName string, deploymentNamespace string) error {
	fmt.Printf("Changing count to %d\n", idealReplicaCt)

	return a.Metrics.ChangeReplicaCount(deploymentNamespace, deploymentName, idealReplicaCt, a.Clientset)
}
