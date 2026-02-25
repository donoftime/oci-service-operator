/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OciVaultKeyShape defines the cryptographic shape for a key managed within the vault.
type OciVaultKeyShape struct {
	// Algorithm is the encryption algorithm for the key (AES, RSA, ECDSA)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=AES;RSA;ECDSA
	Algorithm string `json:"algorithm"`

	// Length is the key length in bytes (AES: 16/24/32, RSA: 256/384/512, ECDSA: 32/48/66)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum:=16
	Length int `json:"length"`
}

// OciVaultKeySpec defines an optional key to create and manage within the vault.
type OciVaultKeySpec struct {
	// KeyId is the OCID of an existing key to bind to (optional; if omitted, a new key is created)
	KeyId OCID `json:"id,omitempty"`

	// DisplayName is a user-friendly name for the key
	// +kubebuilder:validation:Required
	DisplayName string `json:"displayName"`

	// KeyShape defines the cryptographic properties of the key
	// +kubebuilder:validation:Required
	KeyShape OciVaultKeyShape `json:"keyShape"`
}

// OciVaultSpec defines the desired state of OciVault
type OciVaultSpec struct {
	// VaultId is the OCID of an existing Vault to bind to (optional; if omitted, a new vault is created)
	VaultId OCID `json:"id,omitempty"`

	// CompartmentId is the OCID of the compartment in which to create the Vault
	// +kubebuilder:validation:Required
	CompartmentId OCID `json:"compartmentId"`

	// DisplayName is a user-friendly name for the Vault
	// +kubebuilder:validation:Required
	DisplayName string `json:"displayName"`

	// VaultType is the type of vault (DEFAULT or VIRTUAL_PRIVATE)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=DEFAULT;VIRTUAL_PRIVATE
	VaultType string `json:"vaultType"`

	// Key is an optional key to create and manage within the vault
	Key *OciVaultKeySpec `json:"key,omitempty"`

	TagResources `json:",inline,omitempty"`
}

// OciVaultStatus defines the observed state of OciVault
type OciVaultStatus struct {
	OsokStatus OSOKStatus `json:"status"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="DisplayName",type="string",JSONPath=".spec.displayName",priority=1
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status.conditions[-1].type",description="status of the OciVault",priority=0
// +kubebuilder:printcolumn:name="Ocid",type="string",JSONPath=".status.status.ocid",description="Ocid of the OciVault",priority=1
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",priority=0

// OciVault is the Schema for the ocivaults API
type OciVault struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OciVaultSpec   `json:"spec,omitempty"`
	Status OciVaultStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OciVaultList contains a list of OciVault
type OciVaultList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OciVault `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OciVault{}, &OciVaultList{})
}
