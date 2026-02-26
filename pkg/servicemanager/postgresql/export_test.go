/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package postgresql

import "github.com/oracle/oci-go-sdk/v65/psql"

// GetCredentialMapForTest exports getCredentialMap for unit testing.
func GetCredentialMapForTest(dbSystem psql.DbSystem) map[string][]byte {
	return getCredentialMap(dbSystem)
}

// ExportSetClientForTest sets the OCI client on the service manager for unit testing.
func ExportSetClientForTest(m *PostgresDbSystemServiceManager, c PostgresClientInterface) {
	m.ociClient = c
}
