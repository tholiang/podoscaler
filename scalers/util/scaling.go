package util

import (
	"context"

	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	kube_client "k8s.io/client-go/kubernetes"
)

func ChangeReplicaCount(namespace string, deploymentName string, replicaCt int, clientset kube_client.Interface) error {
	hsr := HorizontalScaleRequest{
		DeploymentNamespace: namespace,
		DeploymentName:      deploymentName,
		Replicas:            int32(replicaCt),
	}
	return hScaleFromHSR(clientset, hsr)
}

func hScaleFromHSR(clientset kube_client.Interface, req HorizontalScaleRequest) error {
	// create patch with number of replicas
	patch, err := create_hpatch(req.Replicas)
	if err != nil {
		return err
	}

	// patch deployment/scale resource for given deployment
	// derived from kubectl example: https://kubernetes.io/docs/reference/kubectl/generated/kubectl_patch/
	_, err = clientset.AppsV1().Deployments(req.DeploymentNamespace).Patch(context.TODO(), req.DeploymentName, k8stypes.MergePatchType, patch, metav1.PatchOptions{}, "scale")
	if err != nil {
		return err
	}

	// check every 500ms for 30s or until all replicas are ready
	err = wait.PollUntilContextTimeout(context.TODO(), 500*time.Millisecond, 30*time.Second, true, func(ctx context.Context) (bool, error) {
		deployment, err := clientset.AppsV1().Deployments(req.DeploymentNamespace).Get(context.TODO(), req.DeploymentName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return deployment.Status.ReadyReplicas == req.Replicas, nil
	})
	if err != nil {
		return err
	}

	return nil
}

func VScale(clientset kube_client.Interface, podname string, containername string, cpurequests string) error {
	// create patch with number of replicas
	patch, err := create_vpatch(containername, cpurequests, "100") // limit arbitrarily set to 100 CPU, idk if this works
	if err != nil {
		return err
	}

	// patch pods/resize resource for given deployment
	// derived from kubectl example: https://kubernetes.io/docs/tasks/configure-pod-container/resize-container-resources/
	// I dont really get patch types but this only works with strategic
	_, err = clientset.CoreV1().Pods("default").Patch(context.TODO(), podname, k8stypes.StrategicMergePatchType, patch, metav1.PatchOptions{}, "resize")
	if err != nil {
		return err
	}

	return nil
}

func DeletePod(clientset kube_client.Interface, podname string, namespace string) error {
	err := clientset.CoreV1().Pods(namespace).Delete(context.TODO(), podname, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	return nil
}
