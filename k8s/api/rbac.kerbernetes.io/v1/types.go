package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="LDAP Group",type=string,JSONPath=`.spec.ldapGroupDN`,description="The LDAP group distinguished name"
// +kubebuilder:printcolumn:name="Binding Kind",type=string,JSONPath=`.spec.bindings[0].kind`,description="Kind of the first binding"
// +kubebuilder:printcolumn:name="Binding Name",type=string,JSONPath=`.spec.bindings[0].name`,description="Name of the first binding"

// LdapGroupBinding is the Schema for ldapgroupbinding API
type LdapGroupBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              LdapGroupBindingSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LdapGroupBindingList contains a list of LdapGroupBinding
type LdapGroupBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []LdapGroupBinding `json:"items"`
}

type LdapGroupBindingSpec struct {
	LdapGroupDN string                 `json:"ldapGroupDN"`
	Bindings    []LdapGroupBindingItem `json:"bindings"`
}

type LdapGroupBindingItem struct {
	// kind is the kind of the resource to bind to the LDAP group.
	// +kubebuilder:validation:Enum=ClusterRole;Role
	Kind string `json:"kind"`
	// name is the name of the resource to bind to the LDAP group.
	Name string `json:"name"`
	// namespace is the namespace of the resource to bind to the LDAP group.
	// This field is required if the kind is Role.
	// +optional
	Namespace string `json:"namespace,omitempty"`
	// apiGroup is the API group of the resource to bind to the LDAP group.
	ApiGroup string `json:"apiGroup"`
}