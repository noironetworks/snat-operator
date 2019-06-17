// +build !ignore_autogenerated

// Code generated by openapi-gen. DO NOT EDIT.

// This file was autogenerated by openapi-gen. Do not edit it manually!

package v1

import (
	spec "github.com/go-openapi/spec"
	common "k8s.io/kube-openapi/pkg/common"
)

func GetOpenAPIDefinitions(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
	return map[string]common.OpenAPIDefinition{
		"github.com/gaurav-dalvi/snat-operator/pkg/apis/aci/v1.SnatGlobalInfo":       schema_pkg_apis_aci_v1_SnatGlobalInfo(ref),
		"github.com/gaurav-dalvi/snat-operator/pkg/apis/aci/v1.SnatGlobalInfoSpec":   schema_pkg_apis_aci_v1_SnatGlobalInfoSpec(ref),
		"github.com/gaurav-dalvi/snat-operator/pkg/apis/aci/v1.SnatGlobalInfoStatus": schema_pkg_apis_aci_v1_SnatGlobalInfoStatus(ref),
		"github.com/gaurav-dalvi/snat-operator/pkg/apis/aci/v1.SnatLocalInfo":        schema_pkg_apis_aci_v1_SnatLocalInfo(ref),
		"github.com/gaurav-dalvi/snat-operator/pkg/apis/aci/v1.SnatLocalInfoSpec":    schema_pkg_apis_aci_v1_SnatLocalInfoSpec(ref),
		"github.com/gaurav-dalvi/snat-operator/pkg/apis/aci/v1.SnatLocalInfoStatus":  schema_pkg_apis_aci_v1_SnatLocalInfoStatus(ref),
		"github.com/gaurav-dalvi/snat-operator/pkg/apis/aci/v1.SnatPolicy":           schema_pkg_apis_aci_v1_SnatPolicy(ref),
		"github.com/gaurav-dalvi/snat-operator/pkg/apis/aci/v1.SnatPolicySpec":       schema_pkg_apis_aci_v1_SnatPolicySpec(ref),
		"github.com/gaurav-dalvi/snat-operator/pkg/apis/aci/v1.SnatPolicyStatus":     schema_pkg_apis_aci_v1_SnatPolicyStatus(ref),
	}
}

func schema_pkg_apis_aci_v1_SnatGlobalInfo(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "SnatGlobalInfo is the Schema for the snatglobalinfos API",
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"metadata": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"),
						},
					},
					"spec": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/gaurav-dalvi/snat-operator/pkg/apis/aci/v1.SnatGlobalInfoSpec"),
						},
					},
					"status": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/gaurav-dalvi/snat-operator/pkg/apis/aci/v1.SnatGlobalInfoStatus"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"github.com/gaurav-dalvi/snat-operator/pkg/apis/aci/v1.SnatGlobalInfoSpec", "github.com/gaurav-dalvi/snat-operator/pkg/apis/aci/v1.SnatGlobalInfoStatus", "k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"},
	}
}

func schema_pkg_apis_aci_v1_SnatGlobalInfoSpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "SnatGlobalInfoSpec defines the desired state of SnatGlobalInfo",
				Properties:  map[string]spec.Schema{},
			},
		},
		Dependencies: []string{},
	}
}

func schema_pkg_apis_aci_v1_SnatGlobalInfoStatus(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "SnatGlobalInfoStatus defines the observed state of SnatGlobalInfo",
				Properties:  map[string]spec.Schema{},
			},
		},
		Dependencies: []string{},
	}
}

func schema_pkg_apis_aci_v1_SnatLocalInfo(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "SnatLocalInfo is the Schema for the snatlocalinfos API",
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"metadata": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"),
						},
					},
					"spec": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/gaurav-dalvi/snat-operator/pkg/apis/aci/v1.SnatLocalInfoSpec"),
						},
					},
					"status": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/gaurav-dalvi/snat-operator/pkg/apis/aci/v1.SnatLocalInfoStatus"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"github.com/gaurav-dalvi/snat-operator/pkg/apis/aci/v1.SnatLocalInfoSpec", "github.com/gaurav-dalvi/snat-operator/pkg/apis/aci/v1.SnatLocalInfoStatus", "k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"},
	}
}

func schema_pkg_apis_aci_v1_SnatLocalInfoSpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "SnatLocalInfoSpec defines the desired state of SnatLocalInfo",
				Properties:  map[string]spec.Schema{},
			},
		},
		Dependencies: []string{},
	}
}

func schema_pkg_apis_aci_v1_SnatLocalInfoStatus(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "SnatLocalInfoStatus defines the observed state of SnatLocalInfo",
				Properties:  map[string]spec.Schema{},
			},
		},
		Dependencies: []string{},
	}
}

func schema_pkg_apis_aci_v1_SnatPolicy(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "SnatPolicy is the Schema for the snatpolicies API",
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"metadata": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"),
						},
					},
					"spec": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/gaurav-dalvi/snat-operator/pkg/apis/aci/v1.SnatPolicySpec"),
						},
					},
					"status": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/gaurav-dalvi/snat-operator/pkg/apis/aci/v1.SnatPolicyStatus"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"github.com/gaurav-dalvi/snat-operator/pkg/apis/aci/v1.SnatPolicySpec", "github.com/gaurav-dalvi/snat-operator/pkg/apis/aci/v1.SnatPolicyStatus", "k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"},
	}
}

func schema_pkg_apis_aci_v1_SnatPolicySpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "SnatPolicySpec defines the desired state of SnatPolicy",
				Properties:  map[string]spec.Schema{},
			},
		},
		Dependencies: []string{},
	}
}

func schema_pkg_apis_aci_v1_SnatPolicyStatus(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "SnatPolicyStatus defines the observed state of SnatPolicy",
				Properties:  map[string]spec.Schema{},
			},
		},
		Dependencies: []string{},
	}
}
