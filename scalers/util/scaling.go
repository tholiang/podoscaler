package util

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	kube_client "k8s.io/client-go/kubernetes"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

func HScale(clientset kube_client.Interface, deploymentnamespace string, deploymentname string, numreplicas int32) error {
	// create patch with number of replicas
	patch, err := create_hpatch(numreplicas)
	if err != nil {
		return err
	}

	// patch deployment/scale resource for given deployment
	// derived from kubectl example: https://kubernetes.io/docs/reference/kubectl/generated/kubectl_patch/
	_, err = clientset.AppsV1().Deployments(deploymentnamespace).Patch(context.TODO(), deploymentname, k8stypes.MergePatchType, patch, metav1.PatchOptions{}, "scale")
	if err != nil {
		return err
	}

	return nil
}

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

	return nil
}

func MovePods(clientset kube_client.Interface, deploymentnamespace string, deploymentname string, podsToMove []string) error {
	numPods, err := GetReplicaCt(clientset, deploymentname, deploymentnamespace)
	if err != nil {
		return err
	}

	err = ChangeReplicaCount(deploymentnamespace, deploymentname, numPods+len(podsToMove), clientset)
	if err != nil {
		return fmt.Errorf("no space to move pods")
	}

	err = nil
	for _, podName := range podsToMove {
		delerr := clientset.CoreV1().Pods(deploymentnamespace).Delete(context.TODO(), podName, metav1.DeleteOptions{})
		if delerr != nil {
			err = fmt.Errorf("failed to delete all pods")
		}
	}

	return err
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

func VScalePod(pod *v1beta1.PodMetrics, SCALE_MULTIPLIER float64, clientset kube_client.Interface) {
	usage := pod.Containers[0].Usage.Cpu().MilliValue()
	newRequest := int(float64(usage) * SCALE_MULTIPLIER)
	if newRequest < 10 {
		newRequest = 10
	}

	fmt.Printf("Current req: %d, new req: %d\n", usage, newRequest)
	vsr := VerticalScaleRequest{
		PodNamespace:  pod.GetNamespace(),
		PodName:       pod.GetName(),
		ContainerName: pod.Containers[0].Name,
		CpuRequests:   fmt.Sprintf("%dm", newRequest),
		CpuLimits:     fmt.Sprintf("%dm", newRequest),
	}
	vScaleFromVSR(clientset, vsr)
}

func vScaleFromVSR(clientset kube_client.Interface, req VerticalScaleRequest) {
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

func GetReplicaCt(clientset kube_client.Interface, deploymentName string, namespace string) (int, error) {
	podList, err := GetPodList(clientset, deploymentName, namespace)
	if err != nil {
		return -1, fmt.Errorf("failed to list pods: %w", err)
	}

	return len(podList.Items), nil
}
