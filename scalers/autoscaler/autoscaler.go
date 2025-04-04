package main

import (
	"fmt"
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

const SCALE_UP_MULTIPLIER = 1.2
const SCALE_DOWN_MULTIPLIER = 0.8
const MAX_APS = 30 // profiled per deployment
const MIN_APS = 10 // CPU millivalue

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

	for {
		// if SLO violated AND utilization of total CPU-requests is high
		//      - if every pod is at maxAPS or each pod’s node has no more available CPU
		hScale("deploymentName", "namespace", 1)
		//      - else
		vScaleUp("deploymentName", "namespace")

		// else if latency is too low OR utilization of total CPU-requests is low
		//      - if every pod is at minAPS
		hScale("deploymentName", "namespace", -1)
		//      - else
		vScaleDown("deploymentName", "namespace")

		time.Sleep(5 * time.Second)
	}
}

// in-place scale up smallest pod in deployment
func vScaleUp(deploymentName string, namespace string) {
	pod, err := util.GetSmallestPodOfDeployment(clientset, metrics_clientset, deploymentName, namespace)
	if err != nil {
		panic(err.Error())
	}

	util.VScalePod(pod, SCALE_UP_MULTIPLIER, clientset)
}

// in-place scale down largest pod in deployment
func vScaleDown(deploymentName string, namespace string) {
	pod, err := util.GetLargestPodOfDeployment(clientset, metrics_clientset, deploymentName, namespace)
	if err != nil {
		panic(err.Error())
	}

	util.VScalePod(pod, SCALE_DOWN_MULTIPLIER, clientset)
}

// add or remove `delta` amount of replicas in deployment
func hScale(deploymentName string, namespace string, delta int) {
	replicaCt, err := util.GetReplicaCt(clientset, deploymentName, namespace)
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("Current replica count %d. Changing count to %d\n", replicaCt, replicaCt + delta)
	util.ChangeReplicaCount(namespace, deploymentName, replicaCt + delta, clientset)
}