/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package apigateway

import "github.com/oracle/oci-go-sdk/v65/apigateway"

// ExportSetGatewayClientForTest sets the OCI client on the gateway service manager for unit testing.
func ExportSetGatewayClientForTest(m *GatewayServiceManager, c GatewayClientInterface) {
	m.ociClient = c
}

// ExportSetDeploymentClientForTest sets the OCI client on the deployment service manager for unit testing.
func ExportSetDeploymentClientForTest(m *DeploymentServiceManager, c DeploymentClientInterface) {
	m.ociClient = c
}

// ExportGetGatewayCredentialMap exports getGatewayCredentialMap for unit testing.
func ExportGetGatewayCredentialMap(gw apigateway.Gateway) map[string][]byte {
	return getGatewayCredentialMap(gw)
}
