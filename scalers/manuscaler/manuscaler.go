package manuscaler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"../scalerutil"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	kube_client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metrics_client "k8s.io/metrics/pkg/client/clientset/versioned"
)

/* --- GLOBAL VARS --- */
var clientset kube_client.Interface
var metrics_clientset *metrics_client.Clientset

// test response
func index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "wsg")
}

// horizontal: scale the number of pods of some deployment
func hscalereq(w http.ResponseWriter, r *http.Request) {
	// read request body into HorizontalScaleRequest object
	hsr := HorizontalScaleRequest{}
	b := make([]byte, r.ContentLength)
	r.Body.Read(b)
	err := json.Unmarshal(b, &hsr)
	if err != nil {
		fmt.Fprint(w, err)
		return
	}

	hscale(clientset, hsr.DeploymentNamespace, hsr.DeploymentName, hsr.NumReplicas)
	
	fmt.Fprintf(w, "done")
}

// vertical: scale the resource allocation of some pod
func vscalereq(w http.ResponseWriter, r *http.Request) {
	// read request body into VerticalScaleRequest object
	vsr := VerticalScaleRequest{}
	b := make([]byte, r.ContentLength)
	r.Body.Read(b)
	err := json.Unmarshal(b, &vsr)
	if err != nil {
		fmt.Fprint(w, err)
		return
	}

<<<<<<< HEAD:manuscaler/manuscaler.go
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
=======
	podMetricsList := GetPodMetrics(vsr.PodName)
	fmt.Printf("items len: %d\n", len(podMetricsList.Items))
	for _, v := range podMetricsList.Items {
		fmt.Printf("%s\n", v.GetName())
		fmt.Printf("%s\n", v.GetNamespace())
		fmt.Printf("%vm\n", v.Containers[0].Usage.Cpu().MilliValue())
		fmt.Printf("%vMi\n", v.Containers[0].Usage.Memory().Value()/(1024*1024))
	}

	vscale(clientset, vsr.PodName, vsr.ContainerName, vsr.CpuRequests, vsr.CpuLimits)
>>>>>>> 6131abce630d278e5a08b3db53d3bd687a021850:scalers/manuscaler/manuscaler.go

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

	metrics_clientset, err = metrics_client.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	/* --- HTTPS SERVER INIT ---  */
	http.HandleFunc("/", index)
	http.HandleFunc("/hscale", hscalereq)
	http.HandleFunc("/vscale", vscalereq)
	http.ListenAndServe(":3001", nil)
}
