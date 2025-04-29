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

const DEFAULT_DEPLOYMENT_NAMESPACE = "default"
const DEFAULT_MAPS = 500              // in millicpus
const DEFAULT_LATENCY_THRESHOLD = 100 // in milliseconds

type Autoscaler struct {
	PrometheusUrl                 string
	MinNodeAvailabilityThreshold  float64
	DownscaleUtilizationThreshold float64
	DeploymentNamespace           string
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
	deployments, err := a.Metrics.GetAllDeploymentsFromNamespace(a.Clientset, a.DeploymentNamespace)
	if err != nil {
		fmt.Printf("Failed to get deployments: %s\n", err.Error())
		return err
	}

	for _, deployment := range deployments.Items {
		deploymentName := deployment.Name
		fmt.Printf("Processing deployment: %s\n", deploymentName)

		podList, err := a.Metrics.GetReadyPodListForDeployment(a.Clientset, deploymentName, a.DeploymentNamespace)
		podList, err := util.GetControlledPods(clientset)
		if err != nil {
			fmt.Printf("Failed to get pod list for deployment %s: %s\n", deploymentName, err.Error())
			continue
		}

		utilization, alloc, err := a.Metrics.GetDeploymentUtilAndAlloc(a.Clientset, a.MetricsClientset, deploymentName, a.DeploymentNamespace, podList)
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
				a.hScale(idealReplicaCt, deploymentName)
				a.vScaleTo(newRequests, deploymentName)
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
					a.hScale(idealReplicaCt+1, deploymentName)
					a.Metrics.DeletePod(a.Clientset, pod.Name, a.DeploymentNamespace)
					a.hScale(idealReplicaCt, deploymentName)
				}
			}

			if hasNoCongested {
				fmt.Printf("Deployment %s: External bottleneck detected; doing nothing\n", deploymentName)
				return nil
			} else {
				a.vScaleTo(newRequests, deploymentName)
			}
		} else if utilPercent < a.DownscaleUtilizationThreshold {
			if idealReplicaCt < numPods {
				a.hScale(idealReplicaCt, deploymentName)
			}

			hysteresisMargin := 1 / a.DownscaleUtilizationThreshold
			newRequests = int64(math.Ceil(float64(newRequests) * hysteresisMargin))
			a.vScaleTo(newRequests, deploymentName)
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
func (a *Autoscaler) vScaleTo(millis int64, deploymentName string) error {
	podList, err := a.Metrics.GetReadyPodListForDeployment(a.Clientset, deploymentName, a.DeploymentNamespace)
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
func (a *Autoscaler) hScale(idealReplicaCt int, deploymentName string) error {
	fmt.Printf("Changing count to %d\n", idealReplicaCt)

	return a.Metrics.ChangeReplicaCount(a.DeploymentNamespace, deploymentName, idealReplicaCt, a.Clientset)
}
