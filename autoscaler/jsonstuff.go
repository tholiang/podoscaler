package main

import (
	"encoding/json"
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
	Limits   VerticalPatchResourceSpec `json:"limits"`
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

func create_vpatch(containername string, cpurequests string, cpulimits string) ([]byte, error) {
	containerspec := VerticalPatchSpecContainer{
		Name: containername,
		Resources: VerticalPatchContainerResources{
			Requests: VerticalPatchResourceSpec{cpurequests},
			Limits:   VerticalPatchResourceSpec{cpulimits},
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
	CpuLimits     string `json:"cpulimits"` // we can get rid of this ig
}
