/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package postgresql

import (
	"context"
	"fmt"
	"strconv"

	"github.com/oracle/oci-go-sdk/v65/psql"
)

func (c *PostgresDbSystemServiceManager) addToSecret(ctx context.Context, namespace string, dbSystemName string,
	dbSystem psql.DbSystem) (bool, error) {

	c.Log.InfoLog("Creating the PostgresDbSystem connection secret")
	credMap := getCredentialMap(dbSystem)

	c.Log.InfoLog(fmt.Sprintf("Creating secret for PostgresDbSystem %s in namespace %s", dbSystemName, namespace))
	return c.CredentialClient.CreateSecret(ctx, dbSystemName, namespace, nil, credMap)
}

func getCredentialMap(dbSystem psql.DbSystem) map[string][]byte {
	credMap := make(map[string][]byte)

	if dbSystem.Id != nil {
		credMap["id"] = []byte(*dbSystem.Id)
	}
	if dbSystem.DisplayName != nil {
		credMap["displayName"] = []byte(*dbSystem.DisplayName)
	}

	// Extract primary endpoint from NetworkDetails if available
	if dbSystem.NetworkDetails != nil && dbSystem.NetworkDetails.PrimaryDbEndpointPrivateIp != nil {
		credMap["primaryEndpoint"] = []byte(*dbSystem.NetworkDetails.PrimaryDbEndpointPrivateIp)
	}

	// Default PostgreSQL port
	credMap["port"] = []byte(strconv.Itoa(5432))

	return credMap
}
