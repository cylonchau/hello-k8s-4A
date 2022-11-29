package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	v1admission "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	appv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

type patch struct {
	Op    string            `json:"op"`
	Path  string            `json:"path"`
	Value map[string]string `json:"value"`
}

func serveAdmission(w http.ResponseWriter, r *http.Request) {

	var body []byte
	if data, err := ioutil.ReadAll(r.Body); err == nil {
		body = data
	}
	klog.V(4).Info(fmt.Sprintf("receive request: %v....", string(body)[:130]))
	if len(body) == 0 {
		klog.Error(fmt.Sprintf("admission request body is empty"))
		http.Error(w, fmt.Errorf("admission request body is empty").Error(), http.StatusBadRequest)
		return
	}
	var admission v1admission.AdmissionReview
	codefc := serializer.NewCodecFactory(runtime.NewScheme())
	decoder := codefc.UniversalDeserializer()
	_, _, err := decoder.Decode(body, nil, &admission)

	if err != nil {
		msg := fmt.Sprintf("Request could not be decoded: %v", err)
		klog.Error(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	if admission.Request == nil {
		klog.Error(fmt.Sprintf("admission review can't be used: Request field is nil"))
		http.Error(w, fmt.Errorf("admission review can't be used: Request field is nil").Error(), http.StatusBadRequest)
		return
	}

	switch strings.Split(r.RequestURI, "?")[0] {
	case "/mutate":
		req := admission.Request
		var admissionResp v1admission.AdmissionReview
		admissionResp.APIVersion = admission.APIVersion
		admissionResp.Kind = admission.Kind
		klog.Infof("AdmissionReview for Kind=%v, Namespace=%v Name=%v UID=%v Operation=%v",
			req.Kind.Kind, req.Namespace, req.Name, req.UID, req.Operation)
		switch req.Kind.Kind {
		case "Deployment":
			var (
				respstr []byte
				err     error
				deploy  appv1.Deployment
			)
			if err = json.Unmarshal(req.Object.Raw, &deploy); err != nil {
				respStructure := v1admission.AdmissionResponse{Result: &metav1.Status{
					Message: fmt.Sprintf("could not unmarshal resouces review request: %v", err),
					Code:    http.StatusInternalServerError,
				}}
				klog.Error(fmt.Sprintf("could not unmarshal resouces review request: %v", err))
				if respstr, err = json.Marshal(respStructure); err != nil {
					klog.Error(fmt.Errorf("could not unmarshal resouces review response: %v", err))
					http.Error(w, fmt.Errorf("could not unmarshal resouces review response: %v", err).Error(), http.StatusInternalServerError)
					return
				}
				http.Error(w, string(respstr), http.StatusBadRequest)
				return
			}

			current_annotations := deploy.GetAnnotations()
			pl := []patch{}
			for k, v := range current_annotations {
				pl = append(pl, patch{
					Op:   "add",
					Path: "/metadata/annotations",
					Value: map[string]string{
						k: v,
					},
				})
			}
			pl = append(pl, patch{
				Op:   "add",
				Path: "/metadata/annotations",
				Value: map[string]string{
					deploy.Name + "/Allow": "true",
				},
			})

			annotationbyte, err := json.Marshal(pl)

			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			respStructure := &v1admission.AdmissionResponse{
				UID:     req.UID,
				Allowed: true,
				Patch:   annotationbyte,
				PatchType: func() *v1admission.PatchType {
					t := v1admission.PatchTypeJSONPatch
					return &t
				}(),
				Result: &metav1.Status{
					Message: fmt.Sprintf("could not unmarshal resouces review request: %v", err),
					Code:    http.StatusOK,
				},
			}
			admissionResp.Response = respStructure

			klog.Infof("sending response: %s....", admissionResp.Response.String()[:130])
			respByte, err := json.Marshal(admissionResp)
			if err != nil {
				klog.Errorf("Can't encode response messages: %v", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			klog.Infof("prepare to write response...")
			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write(respByte); err != nil {
				klog.Errorf("Can't write response: %v", err)
				http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
			}

		default:
			klog.Error(fmt.Sprintf("unsupport resouces review request type"))
			http.Error(w, "unsupport resouces review request type", http.StatusBadRequest)
		}

	case "/validate":
		req := admission.Request
		var admissionResp v1admission.AdmissionReview
		admissionResp.APIVersion = admission.APIVersion
		admissionResp.Kind = admission.Kind
		klog.Infof("AdmissionReview for Kind=%v, Namespace=%v Name=%v UID=%v Operation=%v",
			req.Kind.Kind, req.Namespace, req.Name, req.UID, req.Operation)
		var (
			deploy  appv1.Deployment
			respstr []byte
		)
		switch req.Kind.Kind {
		case "Deployment":
			if err = json.Unmarshal(req.Object.Raw, &deploy); err != nil {
				respStructure := v1admission.AdmissionResponse{Result: &metav1.Status{
					Message: fmt.Sprintf("could not unmarshal resouces review request: %v", err),
					Code:    http.StatusInternalServerError,
				}}
				klog.Error(fmt.Sprintf("could not unmarshal resouces review request: %v", err))
				if respstr, err = json.Marshal(respStructure); err != nil {
					klog.Error(fmt.Errorf("could not unmarshal resouces review response: %v", err))
					http.Error(w, fmt.Errorf("could not unmarshal resouces review response: %v", err).Error(), http.StatusInternalServerError)
					return
				}
				http.Error(w, string(respstr), http.StatusBadRequest)
				return
			}
		}
		al := deploy.GetAnnotations()
		respStructure := v1admission.AdmissionResponse{
			UID: req.UID,
		}
		if al[fmt.Sprintf("%s/Allow", deploy.Name)] == "true" {
			respStructure.Allowed = true
			respStructure.Result = &metav1.Status{
				Code: http.StatusOK,
			}
		} else {
			respStructure.Allowed = false
			respStructure.Result = &metav1.Status{
				Code: http.StatusForbidden,
				Reason: func() metav1.StatusReason {
					return metav1.StatusReasonForbidden
				}(),
				Message: fmt.Sprintf("the resource %s couldn't to allow entry.", deploy.Kind),
			}
		}

		admissionResp.Response = &respStructure

		klog.Infof("sending response: %s....", admissionResp.Response.String()[:130])
		respByte, err := json.Marshal(admissionResp)
		if err != nil {
			klog.Errorf("Can't encode response messages: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		klog.Infof("prepare to write response...")
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(respByte); err != nil {
			klog.Errorf("Can't write response: %v", err)
			http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
		}
	}
}
