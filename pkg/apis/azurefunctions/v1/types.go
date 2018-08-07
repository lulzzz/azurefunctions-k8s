package v1

import (
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AzureFunction struct {
	meta_v1.TypeMeta   `json:",inline"`
	meta_v1.ObjectMeta `json:"metadata,omitempty"`
	Spec               AzureFunctionSpec `json:"spec"`
}

type AzureFunctionSpec struct {
	Image        string `json:"image"`
	AccessPolicy string `json:"accessPolicy"`
	Min          *int32 `json:"min"`
	Max          *int32 `json:"max"`
	IngressRoute string `json:"ingressRoute"`
	URL          string `json: "url"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AzureFunctionList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata"`
	Items            []AzureFunction `json:"items"`
}
