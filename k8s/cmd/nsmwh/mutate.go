package main

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
	"k8s.io/api/admission/v1beta1"
	v1 "k8s.io/api/core/v1"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
)

func (s *nsmWebhook) mutate(request *v1beta1.AdmissionRequest) *v1beta1.AdmissionResponse {
	logrus.Info("mutate")
	metaAndSpec, err := getMetaAndSpec(request)
	if err != nil {
		return errorReviewResponse(err)
	}

	init := false
	containers := metaAndSpec.spec.Containers
	if len(metaAndSpec.spec.InitContainers) > 0 {
		containers = metaAndSpec.spec.InitContainers
		init = true
	}

	for _, c := range containers {
		if metaAndSpec.meta.Annotations["ws.networkservicemesh.io"] == "true" {
			//do patch
			id := c.Name
			if metaAndSpec.name != "" {
				id = metaAndSpec.name
			}
			workspace, err := nsmd.RequestWorkspace(s.registry, id)
			patches := append([]patchOperation{}, addVolumeMounts(metaAndSpec.spec, []v1.VolumeMount{
				{
					Name:      "nsm-workspace",
					MountPath: workspace.ClientBaseDir,
					ReadOnly:  false,
				},
			}, init)...)
			t := v1.HostPathDirectoryOrCreate
			patches = append(patches, addVolume(metaAndSpec.spec, []v1.Volume{
				{
					Name: "nsm-workspace",
					VolumeSource: v1.VolumeSource{
						HostPath: &v1.HostPathVolumeSource{
							Path: workspace.HostBasedir + workspace.Workspace,
							Type: &t,
						},
					},
				},
			})...)
			patches = append(patches, addEnv(metaAndSpec.spec, []v1.EnvVar{
				{
					Name:  nsmd.NsmDevicePluginEnv,
					Value: "false",
				},
				{
					Name:  nsmd.NsmServerSocketEnv,
					Value: workspace.ClientBaseDir + workspace.NsmServerSocket,
				},
				{
					Name:  nsmd.NsmClientSocketEnv,
					Value: workspace.ClientBaseDir + workspace.NsmClientSocket,
				},
				{
					Name:  nsmd.WorkspaceEnv,
					Value: workspace.ClientBaseDir,
				},
			}, init)...)
			applyDeploymentKind(patches, request.Kind.Kind)
			bytes, err := json.Marshal(patches)
			if err != nil {
				return errorReviewResponse(err)
			}
			return createReviewResponse(bytes)
		}
	}

	return okReviewResponse()
}
