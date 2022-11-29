package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	autheV1 "k8s.io/api/authentication/v1"
	"k8s.io/klog/v2"
)

func serveAuthentication(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		httpError(w, err)
		return
	}
	klog.V(4).Info("Receiving: ", string(b))

	var tokenReview autheV1.TokenReview
	err = json.Unmarshal(b, &tokenReview)
	if err != nil {
		klog.V(3).Info("Json convert err: ", err)
		httpError(w, err)
		return
	}

	// 提取用户名与密码
	s := strings.SplitN(tokenReview.Spec.Token, "@", 2)
	if len(s) != 2 {
		klog.V(3).Info(fmt.Errorf("badly formatted token: %s", tokenReview.Spec.Token))
		httpError(w, fmt.Errorf("badly formatted token: %s", tokenReview.Spec.Token))
		return
	}
	username, password := s[0], s[1]
	// 查询ldap，验证用户是否合法
	userInfo, err := ldapSearch(username, password)
	if err != nil {
		// 这里不打印日志的原因是 ldapSearch 中打印过了
		return
	}

	// 设置返回的tokenReview
	if userInfo == nil {
		tokenReview.Status.Authenticated = false
	} else {
		tokenReview.Status.Authenticated = true
		tokenReview.Status.User = *userInfo
	}

	b, err = json.Marshal(tokenReview)
	if err != nil {
		klog.V(3).Info("Json convert err: ", err)
		httpError(w, err)
		return
	}
	w.Write(b)
	klog.V(3).Info("Returning: ", string(b))
}
