package rbac

import (
	"context"
	"fmt"

	"github.com/open-policy-agent/opa/rego"
	authoV1 "k8s.io/api/authorization/v1"
	"k8s.io/klog/v2"
)

var module = `package k8s
import future.keywords.in

default allow = false
admin_verbs := {"create", "list", "delete", "update"}
admin_groups := {"admin"}
conf_groups := {"conf"}
conf_verbs := {"list"}
allow  {
	groups := {v | v := input.spec.groups[_]}
	count(admin_groups & groups) > 0
	input.spec.resourceAttributes.verb in admin_verbs
}

allow  {
	groups := {v | v := input.spec.groups[_]}
	count(conf_groups & groups) > 0
	input.spec.resourceAttributes.verb in conf_verbs
}
`

func RBACChek(req *authoV1.SubjectAccessReview) bool {
	fmt.Printf("\n%+v\n", req)
	query, err := rego.New(
		rego.Query("data.k8s.allow"),
		rego.Module("k8s.allow", module),
	).PrepareForEval(context.TODO())

	if err != nil {
		klog.V(4).Info(err)
		return false
	}
	result, err := query.Eval(context.TODO(), rego.EvalInput(req))

	if err != nil {
		klog.V(4).Info("evaluation error:", err)
		return false
	} else if len(result) == 0 {
		klog.V(4).Info("undefined result", err)
		return false
	}
	return result.Allowed()
}
