package util

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	kube_client "k8s.io/client-go/kubernetes"
)

func HScale(clientset kube_client.Interface, deploymentnamespace string, deploymentname string, numreplicas int32) {
	// create patch with number of replicas
	patch, err := create_hpatch(numreplicas)
	if err != nil {
		panic(err)
	}

	// patch deployment/scale resource for given deployment
	// derived from kubectl example: https://kubernetes.io/docs/reference/kubectl/generated/kubectl_patch/
	_, err = clientset.AppsV1().Deployments(deploymentnamespace).Patch(context.TODO(), deploymentname, k8stypes.MergePatchType, patch, metav1.PatchOptions{}, "scale")
	if err != nil {
		panic(err)
	}
}

func HScaleFromHSR(clientset kube_client.Interface, req HorizontalScaleRequest) {
	// create patch with number of replicas
	patch, err := create_hpatch(req.Replicas)
	if err != nil {
		panic(err)
	}

	// patch deployment/scale resource for given deployment
	// derived from kubectl example: https://kubernetes.io/docs/reference/kubectl/generated/kubectl_patch/
	_, err = clientset.AppsV1().Deployments(req.DeploymentNamespace).Patch(context.TODO(), req.DeploymentName, k8stypes.MergePatchType, patch, metav1.PatchOptions{}, "scale")
	if err != nil {
		panic(err)
	}
}

func VScale(clientset kube_client.Interface, podname string, containername string, cpurequests string, cpulimits string) {
	// create patch with number of replicas
	patch, err := create_vpatch(containername, cpurequests, cpulimits)
	if err != nil {
		panic(err)
	}

	// patch pods/resize resource for given deployment
	// derived from kubectl example: https://kubernetes.io/docs/tasks/configure-pod-container/resize-container-resources/
	// I dont really get patch types but this only works with strategic
	_, err = clientset.CoreV1().Pods("default").Patch(context.TODO(), podname, k8stypes.StrategicMergePatchType, patch, metav1.PatchOptions{}, "resize")
	if err != nil {
		panic(err)
	}
}

func VScaleFromVSR(clientset kube_client.Interface, req VerticalScaleRequest) {
	// create patch with number of replicas
	patch, err := create_vpatch(req.ContainerName, req.CpuRequests, req.CpuLimits)
	if err != nil {
		panic(err)
	}

	// patch pods/resize resource for given deployment
	// derived from kubectl example: https://kubernetes.io/docs/tasks/configure-pod-container/resize-container-resources/
	// I dont really get patch types but this only works with strategic
	_, err = clientset.CoreV1().Pods("default").Patch(context.TODO(), req.PodName, k8stypes.StrategicMergePatchType, patch, metav1.PatchOptions{}, "resize")
	if err != nil {
		panic(err)
	}
}
