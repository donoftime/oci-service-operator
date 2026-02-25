/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package queue

import ociqueue "github.com/oracle/oci-go-sdk/v65/queue"

// GetCredentialMapForTest exports getCredentialMap for unit testing.
func GetCredentialMapForTest(q ociqueue.Queue) map[string][]byte {
	return getCredentialMap(q)
}
