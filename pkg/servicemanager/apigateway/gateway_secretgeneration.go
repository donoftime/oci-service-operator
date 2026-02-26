/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package apigateway

import (
	"context"
	"fmt"

	"github.com/oracle/oci-go-sdk/v65/apigateway"
)

func (c *GatewayServiceManager) addToSecret(ctx context.Context, namespace string, gatewayName string,
	gw apigateway.Gateway) (bool, error) {

	c.Log.InfoLog("Creating the ApiGateway connection secret")
	credMap := getGatewayCredentialMap(gw)

	c.Log.InfoLog(fmt.Sprintf("Creating secret for ApiGateway %s in namespace %s", gatewayName, namespace))
	return c.CredentialClient.CreateSecret(ctx, gatewayName, namespace, nil, credMap)
}

func getGatewayCredentialMap(gw apigateway.Gateway) map[string][]byte {
	credMap := make(map[string][]byte)

	if gw.Hostname != nil {
		credMap["hostname"] = []byte(*gw.Hostname)
	}

	return credMap
}
