package main

import (
	"fmt"

	"strings"
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
const SCALE_MULTIPLIER = 1.2
const MAPS = 30 // CPU millivalue

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
		podMetricsList := util.GetPodMetrics(metrics_clientset)

		for _, v := range podMetricsList.Items {
			// TODO: isolate all autoscaled deployments in a unique namespace
			if !strings.HasPrefix(v.GetName(), "testapp") {
				continue
			}

			fmt.Printf("Trying to resize %s...\n", v.GetName())
			container := v.Containers[0]
			usage := container.Usage.Cpu().MilliValue()

			if usage < MAPS {
				// vertical scale
				newRequest := int(float64(usage) * SCALE_MULTIPLIER)
				if newRequest < 10 {
					newRequest = 10
				}
				fmt.Printf("Current req: %d, new req: %d\n", usage, newRequest)
				vsr := util.VerticalScaleRequest{
					PodNamespace:  v.GetNamespace(),
					PodName:       v.GetName(),
					ContainerName: container.Name,
					CpuRequests:   fmt.Sprintf("%dm", newRequest),
					CpuLimits:     fmt.Sprintf("%dm", newRequest),
				}
				util.VScaleFromVSR(clientset, vsr)
			} else {
				// horizontal scale
				deployment, replicaCt := util.GetDeploymentAndReplicaCt(clientset, v.GetNamespace(), v.GetName())
				fmt.Printf("Current replica count %d. Adding one more...\n", replicaCt)
				hsr := util.HorizontalScaleRequest{
					DeploymentNamespace: deployment.GetNamespace(),
					DeploymentName:      deployment.GetName(),
					Replicas: int32(replicaCt + 1),
				}
				util.HScaleFromHSR(clientset, hsr)
			}
			fmt.Println("Successfully resized!")
		}
		time.Sleep(5 * time.Second)
	}
}
