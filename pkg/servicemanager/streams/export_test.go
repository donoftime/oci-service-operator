/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package streams

import "github.com/oracle/oci-go-sdk/v65/streaming"

// ExportSetClientForTest sets the OCI client on the service manager for unit testing.
func ExportSetClientForTest(m *StreamServiceManager, c StreamAdminClientInterface) {
	m.ociClient = c
}

// GetCredentialMapForTest exports getCredentialMap for unit testing.
func GetCredentialMapForTest(stream streaming.Stream) (map[string][]byte, error) {
	return getCredentialMap(stream)
}
