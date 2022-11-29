package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"k8s.io/apiserver/pkg/apis/audit"
	"k8s.io/klog/v2"
)

func serveAudit(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		httpError(w, err)
		return
	}

	var eventList audit.EventList
	err = json.Unmarshal(b, &eventList)
	if err != nil {
		klog.V(3).Info("Json convert err: ", err)
		httpError(w, err)
		return
	}
	for _, event := range eventList.Items {

		// here is your logic

		fmt.Printf("审计ID %s: 用户<%s>, 请求对象<%s>, 操作<%s>, 请求阶段<%s>\n",
			event.AuditID,
			event.User.UID,
			event.RequestURI,
			event.Verb,
			event.Stage,
		)
	}
	w.WriteHeader(http.StatusOK)
}
