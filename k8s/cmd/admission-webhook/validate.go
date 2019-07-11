package main

import (
	"github.com/sirupsen/logrus"
	"k8s.io/api/admission/v1beta1"
)

func (s *nsmAdmissionWebhook) validate(request *v1beta1.AdmissionRequest) *v1beta1.AdmissionResponse {
	logrus.Infof("Validating review for %v", request)
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
	err = validateAnnotationValue(value)
	if err != nil {
		return errorReviewResponse(err)
	}
	err = checkNsmInitDuplication(metaAndSpec.spec)
	if err != nil {
		return errorReviewResponse(err)
	}

	return okReviewResponse()
}
