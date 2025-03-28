package manuscaler

import (
	"fmt"
	"net/http"

	"k8s.io/apimachinery/pkg/api/errors"
	kube_client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

/* --- GLOBAL VARS --- */
var clientset ScalesGetter

func index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello")
}

func setreplicas(w http.ResponseWriter, r *http.Request) {

}

func main() {
	/* --- K8S CLIENT GO CONFIG STUFF --- */
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}



	/* --- HTTPS SERVER INIT ---  */
	http.HandleFunc("/", index)
	http.HandleFunc("/setreplicas", setreplicas)
	http.ListenAndServe(":3001", nil)
}
