/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package vault

import "github.com/oracle/oci-go-sdk/v65/keymanagement"

// GetCredentialMapForTest exports getCredentialMap for unit testing.
func GetCredentialMapForTest(v keymanagement.Vault) map[string][]byte {
	return getCredentialMap(v)
}

// ExportSetVaultClientForTest sets the KMS vault client on the service manager for unit testing.
func ExportSetVaultClientForTest(m *OciVaultServiceManager, c KmsVaultClientInterface) {
	m.ociVaultClient = c
}

// ExportSetManagementClientForTest sets the KMS management client on the service manager for unit testing.
func ExportSetManagementClientForTest(m *OciVaultServiceManager, c KmsManagementClientInterface) {
	m.ociManagementClient = c
}
