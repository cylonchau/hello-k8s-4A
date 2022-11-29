package server

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	authoV1 "k8s.io/api/authorization/v1"
	"k8s.io/klog/v2"

	"github.com/cylonchau/hello-k8s-4A/rbac"
)

func serveAuthorization(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		httpError(w, err)
		return
	}
	klog.V(4).Info("Receied: ", string(b))

	var subjectAccessReview authoV1.SubjectAccessReview
	err = json.Unmarshal(b, &subjectAccessReview)
	if err != nil {
		klog.V(3).Info("Json convert err: ", err)
		httpError(w, err)
		return
	}
	subjectAccessReview.Status.Allowed = rbac.RBACChek(&subjectAccessReview)
	b, err = json.Marshal(subjectAccessReview)
	if err != nil {
		klog.V(3).Info("Json convert err: ", err)
		httpError(w, err)
		return
	}
	w.Write(b)
	klog.V(3).Info("Returning: ", string(b))
}
