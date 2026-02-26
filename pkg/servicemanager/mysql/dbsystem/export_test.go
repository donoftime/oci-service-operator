/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dbsystem

// ExportSetClientForTest sets the OCI client on the service manager for unit testing.
func ExportSetClientForTest(m *DbSystemServiceManager, c MySQLDbSystemClientInterface) {
	m.ociClient = c
}
