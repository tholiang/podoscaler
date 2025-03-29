package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	kube_client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

/* --- GLOBAL VARS --- */
var clientset kube_client.Interface

// test response
func index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "wsg")
}

// horizontal: scale the number of pods of some deployment
func hscale(w http.ResponseWriter, r *http.Request) {
	// read request body into HorizontalScaleRequest object
	hsr := HorizontalScaleRequest{}
	b := make([]byte, r.ContentLength)
	r.Body.Read(b)
	err := json.Unmarshal(b, &hsr)
	if err != nil {
		fmt.Fprint(w, err)
		return
	}

	// create patch with number of replicas
	patch, err := create_hpatch(hsr.Replicas)
	if err != nil {
		fmt.Fprint(w, err)
		return
	}

	// patch deployment/scale resource for given deployment
	// derived from kubectl example: https://kubernetes.io/docs/reference/kubectl/generated/kubectl_patch/
	_, err = clientset.AppsV1().Deployments(hsr.DeploymentNamespace).Patch(context.TODO(), hsr.DeploymentName, k8stypes.MergePatchType, patch, metav1.PatchOptions{}, "scale")
	if err != nil {
		fmt.Fprint(w, err)
		return
	}
	fmt.Fprintf(w, "done")
}

// vertical: scale the resource allocation of some pod
func vscale(w http.ResponseWriter, r *http.Request) {
	// read request body into VerticalScaleRequest object
	vsr := VerticalScaleRequest{}
	b := make([]byte, r.ContentLength)
	r.Body.Read(b)
	err := json.Unmarshal(b, &vsr)
	if err != nil {
		fmt.Fprint(w, err)
		return
	}

	// create patch with number of replicas
	patch, err := create_vpatch(vsr.ContainerName, vsr.CpuRequests, vsr.CpuLimits)
	if err != nil {
		fmt.Fprint(w, err)
		return
	}

	// patch pods/resize resource for given deployment
	// derived from kubectl example: https://kubernetes.io/docs/tasks/configure-pod-container/resize-container-resources/
	// I dont really get patch types but this only works with strategic
	_, err = clientset.CoreV1().Pods("default").Patch(context.TODO(), vsr.PodName, k8stypes.StrategicMergePatchType, patch, metav1.PatchOptions{}, "resize")
	if err != nil {
		fmt.Fprint(w, err)
		return
	}
	fmt.Fprintf(w, "done")
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

	/* --- HTTPS SERVER INIT ---  */
	http.HandleFunc("/", index)
	http.HandleFunc("/hscale", hscale)
	http.HandleFunc("/vscale", vscale)
	http.ListenAndServe(":3001", nil)
}
