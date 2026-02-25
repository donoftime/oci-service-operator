/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package vault

import (
	"context"
	"fmt"

	"github.com/oracle/oci-go-sdk/v65/keymanagement"
)

func (c *OciVaultServiceManager) addToSecret(ctx context.Context, namespace string, vaultName string,
	v keymanagement.Vault) (bool, error) {

	c.Log.InfoLog("Creating the OciVault connection secret")
	credMap := getCredentialMap(v)

	c.Log.InfoLog(fmt.Sprintf("Creating secret for OciVault %s in namespace %s", vaultName, namespace))
	return c.CredentialClient.CreateSecret(ctx, vaultName, namespace, nil, credMap)
}

func getCredentialMap(v keymanagement.Vault) map[string][]byte {
	credMap := make(map[string][]byte)

	if v.Id != nil {
		credMap["id"] = []byte(*v.Id)
	}
	if v.DisplayName != nil {
		credMap["displayName"] = []byte(*v.DisplayName)
	}
	if v.ManagementEndpoint != nil {
		credMap["managementEndpoint"] = []byte(*v.ManagementEndpoint)
	}
	if v.CryptoEndpoint != nil {
		credMap["cryptoEndpoint"] = []byte(*v.CryptoEndpoint)
	}

	return credMap
}
