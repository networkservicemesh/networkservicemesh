package main

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
	"k8s.io/api/admission/v1beta1"
	"k8s.io/api/core/v1"
)

func (s *nsmAdmissionWebhook) mutate(request *v1beta1.AdmissionRequest) *v1beta1.AdmissionResponse {
	logrus.Infof("AdmissionReview for =%v", request)
	if !isSupportKind(request) {
		return okReviewResponse()
	}
	metaAndSpec, err := getMetaAndSpec(request)
	if err != nil {
		return errorReviewResponse(err)
	}
	value, ok := getNsmAnnotationValue(ignoredNamespaces, metaAndSpec)
	if !ok {
		logrus.Infof("Skipping validation for %s/%s due to policy check", metaAndSpec.meta.Namespace, metaAndSpec.meta.Name)
		return okReviewResponse()
	}
	if err = validateAnnotationValue(value); err != nil {
		return errorReviewResponse(err)
	}
	if err = checkNsmInitContainerDuplication(metaAndSpec.spec); err != nil {
		logrus.Error(err)
		return errorReviewResponse(err)
	}
	patch := createInitContainerPatch(value, getInitContainerPatchPath(request))

	ht := v1.HostPathDirectoryOrCreate
	patch = append(patch, addVolume(metaAndSpec.spec.Volumes, []v1.Volume{
		{
			Name: spireSocketVolume,
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: spireSocketPath,
					Type: &ht,
				},
			},
		},
	}, getVolumePatchPath(request))...)

	//append another patches
	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return errorReviewResponse(err)
	}
	logrus.Infof("AdmissionResponse: patch=%v\n", string(patchBytes))
	return createReviewResponse(patchBytes)
}
