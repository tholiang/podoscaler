//go:build manuscaler
// +build manuscaler

package manuscaler

import (
	"encoding/json"
	"fmt"
	"net/http"

	util "github.com/tholiang/podoscaler/scalers/util"

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
	hsr := util.HorizontalScaleRequest{}
	b := make([]byte, r.ContentLength)
	r.Body.Read(b)
	err := json.Unmarshal(b, &hsr)
	if err != nil {
		fmt.Fprint(w, err)
		return
	}

	util.HScale(clientset, hsr.DeploymentNamespace, hsr.DeploymentName, hsr.Replicas)

	fmt.Fprintf(w, "done")
}

// vertical: scale the resource allocation of some pod
func vscalereq(w http.ResponseWriter, r *http.Request) {
	// read request body into VerticalScaleRequest object
	vsr := util.VerticalScaleRequest{}
	b := make([]byte, r.ContentLength)
	r.Body.Read(b)
	err := json.Unmarshal(b, &vsr)
	if err != nil {
		fmt.Fprint(w, err)
		return
	}

	util.VScale(clientset, vsr.PodName, vsr.ContainerName, vsr.CpuRequests, vsr.CpuLimits)

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
