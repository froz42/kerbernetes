package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LdapClusterRoleBinding is the Schema for ldapclusterrolebindings API
type LdapClusterRoleBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec LdapClusterRoleBindingSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LdapClusterRoleBindingList contains a list of LdapClusterRoleBinding
type LdapClusterRoleBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []LdapClusterRoleBinding `json:"items"`
}

// LdapClusterRoleBindingSpec defines the desired state of LdapClusterRoleBinding
type LdapClusterRoleBindingSpec struct {
	LdapGroupDN      string `json:"ldapGroupDN"`
	ClusterRoleRef struct {
		Name string `json:"name"`
	} `json:"clusterRoleRef"`
}
