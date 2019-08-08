package main

import (
	"github.com/sirupsen/logrus"
	"k8s.io/api/admission/v1beta1"
)

func (s *nsmWebhook) validate(request *v1beta1.AdmissionRequest) *v1beta1.AdmissionResponse {
	logrus.Info("validate")
	return okReviewResponse()
}
