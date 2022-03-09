package v1

import (
	"github.com/MagalixCorp/magalix-policy-agent/pkg/domain"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Policy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              domain.Policy `json:"spec"`
}

type PolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Policy `json:"items"`
}

func (p *Policy) deepCopy(out *Policy) {
	out.TypeMeta = p.TypeMeta
	out.ObjectMeta = p.ObjectMeta
	out.Spec = p.Spec
}

// DeepCopyObject returns a copy of policy crd data. Implements the Kuberntes object interface
func (in *Policy) DeepCopyObject() runtime.Object {
	out := Policy{}
	in.deepCopy(&out)

	return &out
}

// DeepCopyObject returns a copy of policy crd list data. Implements the Kuberntes object interface
func (in *PolicyList) DeepCopyObject() runtime.Object {
	out := PolicyList{}
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta

	if in.Items != nil {
		out.Items = make([]Policy, len(in.Items))
		for i := range in.Items {
			in.Items[i].deepCopy(&out.Items[i])
		}
	}

	return &out
}
