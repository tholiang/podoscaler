package main

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	kube_client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metrics_client "k8s.io/metrics/pkg/client/clientset/versioned"
	"time"
	"strings"
)


/* --- GLOBAL VARS --- */
var clientset kube_client.Interface
var metrics_clientset *metrics_client.Clientset
const SCALE_MULTIPLIER = 1.2

// horizontal: scale the number of pods of some deployment
func hscale() {

}

// vertical: scale the resource allocation of some pod
func vscale(vsr VerticalScaleRequest) {
	// create patch with number of replicas
	patch, err := create_vpatch(vsr.ContainerName, vsr.CpuRequests, vsr.CpuLimits)
	if err != nil {
		panic(err.Error())
	}

	// patch pods/resize resource for given deployment
	// derived from kubectl example: https://kubernetes.io/docs/tasks/configure-pod-container/resize-container-resources/
	// I dont really get patch types but this only works with strategic
	_, err = clientset.CoreV1().Pods("default").Patch(context.TODO(), vsr.PodName, k8stypes.StrategicMergePatchType, patch, metav1.PatchOptions{}, "resize")
	if err != nil {
		panic(err.Error())
	}
}

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
		podMetricsList, err := metrics_clientset.MetricsV1beta1().PodMetricses("default").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}

		for _, v := range podMetricsList.Items {
			if !strings.HasPrefix(v.GetName(), "testapp") { continue }
			
			fmt.Printf("Trying to resize %s...\n", v.GetName())
			newRequest := int(float64(v.Containers[0].Usage.Cpu().MilliValue()) * SCALE_MULTIPLIER)
			if newRequest < 10 {
				newRequest = 10
			}
			fmt.Printf("current req: %d, new req: %d\n", v.Containers[0].Usage.Cpu().MilliValue(), newRequest)
			container := v.Containers[0]
			vsr := VerticalScaleRequest {
				PodNamespace: v.GetNamespace(),
				PodName:       v.GetName(),
				ContainerName: container.Name,
				CpuRequests:   fmt.Sprintf("%dm", newRequest),
				CpuLimits:    fmt.Sprintf("%dm", newRequest),
			}
			vscale(vsr)
			fmt.Println("Successfully resized!")
		}
		time.Sleep(5 * time.Second)
	}
}
