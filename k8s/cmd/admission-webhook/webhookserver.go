package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type nsmAdmissionWebhook struct {
	server *http.Server
}

func (s *nsmAdmissionWebhook) serve(w http.ResponseWriter, r *http.Request) {
	body, err := readRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	requestReview, err := parseAdmissionReview(body)
	nsmAdmissionWebhookReview := v1beta1.AdmissionReview{}
	if err != nil {
		nsmAdmissionWebhookReview.Response = &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		if r.URL.Path == validateMethod {
			nsmAdmissionWebhookReview.Response = s.validate(requestReview.Request)
		} else if r.URL.Path == mutateMethod {
			nsmAdmissionWebhookReview.Response = s.mutate(requestReview.Request)
		}
	}
	nsmAdmissionWebhookReview.Response.UID = requestReview.Request.UID
	resp, err := json.Marshal(nsmAdmissionWebhookReview)
	if err != nil {
		logrus.Errorf("Can't encode response: %v", err)
		http.Error(w, fmt.Sprintf(couldNotEncodeReview, err), http.StatusInternalServerError)
	}
	if _, err := w.Write(resp); err != nil {
		logrus.Errorf("Can't write response: %v", err)
		http.Error(w, fmt.Sprintf(couldNotWriteReview, err), http.StatusInternalServerError)
	}
}
