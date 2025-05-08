package util

import (
	"encoding/json"
	"fmt"
)

// theres definitely a better way to do this

/* --- DEFINITIONS --- */
type HorizontalPatchSpec struct {
	Replicas int32 `json:"replicas"`
}

type HorizontalPatch struct {
	Spec HorizontalPatchSpec `json:"spec"`
}

func create_hpatch(replicas int32) ([]byte, error) {
	hp := HorizontalPatch{HorizontalPatchSpec{Replicas: replicas}}
	return json.Marshal(hp)
}

type HorizontalScaleRequest struct {
	DeploymentNamespace string `json:"deploymentnamespace"`
	DeploymentName      string `json:"deploymentname"`
	Replicas            int32  `json:"replicas"`
}

/* --- */
// patch from https://kubernetes.io/docs/tasks/configure-pod-container/resize-container-resources/
type VerticalPatchResourceSpec struct {
	CPU string `json:"cpu"`
}

type VerticalPatchContainerResources struct {
	Requests VerticalPatchResourceSpec `json:"requests"`
}

type VerticalPatchSpecContainer struct {
	Name      string                          `json:"name"`
	Resources VerticalPatchContainerResources `json:"resources"`
}

type VerticalPatchSpec struct {
	Containers []VerticalPatchSpecContainer `json:"containers"`
}

type VerticalPatch struct {
	Spec VerticalPatchSpec `json:"spec"`
}

func create_vpatch(containername string, cpurequests string) ([]byte, error) {
	containerspec := VerticalPatchSpecContainer{
		Name: containername,
		Resources: VerticalPatchContainerResources{
			Requests: VerticalPatchResourceSpec{cpurequests},
		},
	}

	vp := VerticalPatch{
		VerticalPatchSpec{
			[]VerticalPatchSpecContainer{containerspec},
		},
	}
	return json.Marshal(vp)
}

type VerticalScaleRequest struct {
	PodNamespace  string `json:"podnamespace"`
	PodName       string `json:"podname"`
	ContainerName string `json:"containername"`
	CpuRequests   string `json:"cpurequests"`
}

type DeploymentPatchObj struct {
	Operation string                          `json:"op"`
	Path      string                          `json:"path"`
	Value     VerticalPatchContainerResources `json:"value"`
}

type DeploymentPatch []DeploymentPatchObj

func create_deployment_request_patch(containeridx int, cpurequests string) ([]byte, error) {
	dp := DeploymentPatch{
		DeploymentPatchObj{
			Operation: "replace",
			Path:      fmt.Sprintf("/spec/containers/%d/resources", containeridx),
			Value: VerticalPatchContainerResources{
				Requests: VerticalPatchResourceSpec{cpurequests},
			},
		},
	}
	return json.Marshal(dp)
}
