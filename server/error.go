package server

import (
	"fmt"
	"net/http"

	"k8s.io/klog/v2"
)

func httpError(w http.ResponseWriter, err error) {
	err = fmt.Errorf("Error: %v", err)
	w.WriteHeader(http.StatusInternalServerError) // 500
	fmt.Fprintln(w, err)
	klog.V(4).Info("httpcode 500: ", err)
}
