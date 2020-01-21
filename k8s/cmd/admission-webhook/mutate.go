package main

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/admission/v1beta1"
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
		return errorReviewResponse(err)
	}

	annotations := metaAndSpec.meta.GetAnnotations()
	resourceName, ok := annotations["networkservicemesh.io/resourcename"]
	if !ok {
		return errorReviewResponse(errors.Errorf("networkservicemesh.io/resourcename annotation missing, annotations: %v", annotations))
	}
	resourcePrefix, ok := annotations["networkservicemesh.io/resourceprefix"]
	if !ok {
		return errorReviewResponse(errors.Errorf("networkservicemesh.io/resourceprefix annotation missing, annotations: %v", annotations))
	}

	patch := createNsmInitContainerPatch(value, resourcePrefix, resourceName)
	patch = append(patch, createDNSPatch(metaAndSpec, value)...)
	//append another patches
	applyDeploymentKind(patch, request.Kind.Kind)
	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return errorReviewResponse(err)
	}
	logrus.Infof("AdmissionResponse: patch=%v\n", string(patchBytes))

	return createReviewResponse(patchBytes)
}
