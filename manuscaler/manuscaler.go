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

/* --- DEFINITIONS --- */
type HorizontalPatchSpec struct {
	Replicas int32 `json:"replicas"`
}

type HorizontalPatch struct {
	Spec HorizontalPatchSpec `json:"spec"`
}
type HorizontalScaleRequest struct {
	DeploymentNamespace string `json:"deploymentnamespace"`
	DeploymentName      string `json:"deploymentname"`
	Replicas            int32  `json:"replicas"`
}

type VerticalPatch struct {
}

type VerticalScaleRequest struct {
	PodNamespace string
	PodName      string
}

/* --- GLOBAL VARS --- */
var clientset kube_client.Interface

// test response
func index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "wsg")
}

// horizontal: scale the number of pods of some resource
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
	hp := HorizontalPatch{HorizontalPatchSpec{Replicas: hsr.Replicas}}
	patch, err := json.Marshal(hp)
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

func vscale(w http.ResponseWriter, r *http.Request) {
	// decoder := json.NewDecoder(r.Body)
	// var t verticalscalerequest
	// err := decoder.Decode(&t)

	// patch, err := json.Marshal(verticalpatch)
	// if err != nil {
	// 	fmt.Fprint(w, err)
	// 	return
	// }

	// _, err = clientset.CoreV1().Pods("[namespace of pod]").Patch(context.TODO(), "name of pod", k8stypes.JSONPatchType, patch, metav1.PatchOptions{}, "resize")
	// if err != nil {
	// 	fmt.Fprint(w, err)
	// 	return
	// }
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
