/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package networking

// ExportSetVcnClientForTest sets the OCI client on VcnServiceManager for unit testing.
func ExportSetVcnClientForTest(m *OciVcnServiceManager, c VirtualNetworkClientInterface) {
	m.ociClient = c
}

// ExportSetSubnetClientForTest sets the OCI client on SubnetServiceManager for unit testing.
func ExportSetSubnetClientForTest(m *OciSubnetServiceManager, c VirtualNetworkClientInterface) {
	m.ociClient = c
}

// ExportSetInternetGatewayClientForTest sets the OCI client on InternetGatewayServiceManager for unit testing.
func ExportSetInternetGatewayClientForTest(m *OciInternetGatewayServiceManager, c VirtualNetworkClientInterface) {
	m.ociClient = c
}

// ExportSetNatGatewayClientForTest sets the OCI client on NatGatewayServiceManager for unit testing.
func ExportSetNatGatewayClientForTest(m *OciNatGatewayServiceManager, c VirtualNetworkClientInterface) {
	m.ociClient = c
}

// ExportSetServiceGatewayClientForTest sets the OCI client on ServiceGatewayServiceManager for unit testing.
func ExportSetServiceGatewayClientForTest(m *OciServiceGatewayServiceManager, c VirtualNetworkClientInterface) {
	m.ociClient = c
}

// ExportSetDrgClientForTest sets the OCI client on DrgServiceManager for unit testing.
func ExportSetDrgClientForTest(m *OciDrgServiceManager, c VirtualNetworkClientInterface) {
	m.ociClient = c
}
