package main

import (
	"encoding/json"
	"net/http"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"

	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type nsmWebhook struct {
	server   *http.Server
	registry serviceregistry.ServiceRegistry
}

func (s *nsmWebhook) serve(w http.ResponseWriter, r *http.Request) {
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
		if r.URL.Path == "/mutate" {
			nsmAdmissionWebhookReview.Response = s.mutate(requestReview.Request)
		}
		if r.URL.Path == "/validate" {
			nsmAdmissionWebhookReview.Response = s.validate(requestReview.Request)
		}
	}
	nsmAdmissionWebhookReview.Response.UID = requestReview.Request.UID
	resp, err := json.Marshal(nsmAdmissionWebhookReview)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if _, err := w.Write(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
