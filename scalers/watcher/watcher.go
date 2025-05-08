//go:build watcher
// +build watcher

package watcher

import (
	"fmt"
	"os"

	"github.com/tholiang/podoscaler/scalers/util"
	"k8s.io/client-go/kubernetes"
	kube_client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metrics_client "k8s.io/metrics/pkg/client/clientset/versioned"
)

type DeploymentRoundData struct {
	TotalAllocation int64
	TotalUsage      int64
	NumPods         int
}

type NodeRoundData struct {
	Capacity   int64
	Allocation int64
	Usage      int64
}

type RoundData struct {
	Latencies   map[string]float64             // percentile ("p90", "p95", "p99") to latency millis
	Nodes       map[string]NodeRoundData       // name to data
	Deployments map[string]DeploymentRoundData // name to data
}

type Watcher struct {
	PrometheusUrl    string
	Clientset        kube_client.Interface
	MetricsClientset *metrics_client.Clientset

	rounds int64
	data   []RoundData
}

func (w *Watcher) Init() error {
	/* --- CONFIGURATION LOGIC --- */
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}

	// creates the clientset
	w.Clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}
	w.MetricsClientset, err = metrics_client.NewForConfig(config)
	if err != nil {
		return err
	}

	// set env variable for Prometheus service url
	os.Setenv("PROMETHEUS_URL", util.DEFAULT_PROMETHEUS_URL)

	// other
	w.rounds = 0
	w.data = []RoundData{}
	return nil
}

func (w *Watcher) WatchRound() error {
	fmt.Printf("Round %d\n", w.rounds)
	w.rounds++
	var rounddata = RoundData{Latencies: map[string]float64{}, Nodes: map[string]NodeRoundData{}, Deployments: map[string]DeploymentRoundData{}}

	// get node usages
	nodelist, err := util.GetNodeList(w.Clientset)
	if err != nil {
		fmt.Printf("ERROR: Failed to get node list: %s\n", err.Error())
		return err
	}

	for _, node := range nodelist.Items {
		nodeName := node.Name

		allocable, capacity, err := util.GetNodeAllocableAndCapacity(w.Clientset, nodeName)
		if err != nil {
			fmt.Printf("ERROR: Failed to get node metrics for node %s: %s\n", nodeName, err.Error())
			continue
		}

		usage, err := util.GetNodeUsage(w.MetricsClientset, nodeName)
		if err != nil {
			fmt.Printf("ERROR: Failed to get usage for node %s: %s\n", nodeName, err.Error())
			continue
		}

		nodedata := NodeRoundData{Capacity: capacity, Allocation: capacity - allocable, Usage: usage}
		rounddata.Nodes[nodeName] = nodedata
	}

	// get latency
	lb_name, err := util.GetLoadBalancerName(w.Clientset, os.Getenv("AUTOSCALE_NAMESPACE"), os.Getenv("AUTOSCALE_LB"))
	if err != nil {
		fmt.Printf("ERROR: Failed to get load balancer name: %s\n", err.Error())
		return err
	}
	rounddata.Latencies, err = util.GetLatencyCloudwatch(lb_name)
	if err != nil {
		fmt.Printf("ERROR: Failed to get latency: %s\n", err.Error())
		return err
	}

	// Get all deployments in the namespace
	deployments, err := util.GetControlledDeployments(w.Clientset)
	if err != nil {
		fmt.Printf("ERROR: Failed to get deployments: %s\n", err.Error())
		return err
	}

	for _, deployment := range deployments.Items {
		var deploymentdata = DeploymentRoundData{}

		deploymentName := deployment.Name
		deploymentNamespace := deployment.Namespace

		podList, err := util.GetReadyPodListForDeployment(w.Clientset, deploymentName, deploymentNamespace)
		if err != nil {
			fmt.Printf("ERROR: Failed to get pod list for deployment %s: %s\n", deploymentName, err.Error())
			continue
		}

		utilization, alloc, err := util.GetDeploymentUtilAndAlloc(w.Clientset, w.MetricsClientset, deploymentName, deploymentNamespace, podList)
		if err != nil {
			fmt.Printf("ERROR: Failed to get utilization metrics for deployment %s: %s\n", deploymentName, err.Error())
			continue
		}
		deploymentdata.TotalAllocation = alloc
		deploymentdata.TotalUsage = utilization
		numPods := len(podList)
		deploymentdata.NumPods = numPods

		rounddata.Deployments[deploymentName] = deploymentdata
	}

	// print output
	for percentile, latency := range rounddata.Latencies {
		fmt.Printf("percentile %s latency %d\n", percentile, latency)
	}

	for node, data := range rounddata.Nodes {
		fmt.Printf("node %s capacity %d allocation %d usage %d\n", node, data.Capacity, data.Allocation, data.Usage)
	}

	for deployment, data := range rounddata.Deployments {
		fmt.Printf("deployment %s allocation %d usage %d pods %d\n", deployment, data.TotalAllocation, data.TotalUsage, data.NumPods)
	}

	fmt.Println()

	return nil
}
