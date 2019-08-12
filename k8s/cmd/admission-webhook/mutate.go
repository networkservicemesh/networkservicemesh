package main

import (
	"encoding/json"

	v1 "k8s.io/api/core/v1"

	"github.com/sirupsen/logrus"
	"k8s.io/api/admission/v1beta1"
)

func (s *nsmAdmissionWebhook) mutate(request *v1beta1.AdmissionRequest) *v1beta1.AdmissionResponse {
	logrus.Infof("AdmissionReview for: Kind - %v, Resource - %v", request.Kind, request.Resource)
	if !isSupportKind(request) {
		return okReviewResponse()
	}

	metaAndSpec, err := getMetaAndSpec(request)
	if err != nil {
		return errorReviewResponse(err)
	}

	logrus.Infof("Annotations: %v", metaAndSpec.meta.Annotations)

	if isIgnoreNamespace(ignoredNamespaces, metaAndSpec) {
		logrus.Infof("Skip validation for %v for it's in special namespace:%v", metaAndSpec.meta.Name, metaAndSpec.meta.Namespace)
		return okReviewResponse()
	}

	annotations := metaAndSpec.meta.GetAnnotations()
	if annotations == nil {
		logrus.Info("No annotations, skip")
		return okReviewResponse()
	}

	var patch []patchOperation

	_, secure := annotations[securityAnnotationKey]
	if secure {
		logrus.Infof("%v annotation is discovered, preparing security patch...", securityAnnotationKey)

		ht := v1.HostPathDirectoryOrCreate
		patch = append(patch, addVolume(metaAndSpec.spec, v1.Volume{
			Name: spireSocketVolume,
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: spireSocketPath,
					Type: &ht,
				},
			},
		}, getSpecPath(request))...)

		patch = append(patch, addVolumeMounts(metaAndSpec.spec, v1.VolumeMount{
			Name:      spireSocketVolume,
			MountPath: spireSocketPath,
			ReadOnly:  true,
		}, getSpecPath(request))...)
	}

	nsmAnnotationValue, ok := annotations[nsmAnnotationKey]
	if ok {
		logrus.Infof("%v annotation is discovered, preparing init-container patch...", nsmAnnotationKey)

		if err = validateAnnotationValue(nsmAnnotationValue); err != nil {
			return errorReviewResponse(err)
		}

		if err = checkNsmInitContainerDuplication(metaAndSpec.spec); err != nil {
			logrus.Error(err)
			return errorReviewResponse(err)
		}

		patch = append(patch,
			createInitContainerPatch(nsmAnnotationValue, getInitContainerPatchPath(request), secure)...)
	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return errorReviewResponse(err)
	}
	logrus.Infof("AdmissionResponse: patch=%v\n", string(patchBytes))
	return createReviewResponse(patchBytes)
}
