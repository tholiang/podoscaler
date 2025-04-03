package util

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	kube_client "k8s.io/client-go/kubernetes"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/labels"
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

func GetDeploymentAndReplicaCt(clientset kube_client.Interface, namespace string, podName string) (*appsv1.Deployment, int) {
	ctx := context.TODO()

	pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		panic(err)
	}

	for _, ownerRef := range pod.OwnerReferences {
		if ownerRef.Kind == "ReplicaSet" {
			rs, err := clientset.AppsV1().ReplicaSets(namespace).Get(ctx, ownerRef.Name, metav1.GetOptions{})
			if err != nil {
				panic(err)
			}

			for _, rsOwnerRef := range rs.OwnerReferences {
				if rsOwnerRef.Kind == "Deployment" {
					deployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, rsOwnerRef.Name, metav1.GetOptions{})
					if err != nil {
						panic(err)
					}
					
					labelSelector := labels.Set(deployment.Spec.Selector.MatchLabels).AsSelector().String()

					podList, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
						LabelSelector: labelSelector,
					})
					if err != nil {
						panic(err)
					}

					return deployment, len(podList.Items)
				}
			}
		}
	}

	return nil, -1
}