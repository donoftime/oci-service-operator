/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package functions

import ocifunctions "github.com/oracle/oci-go-sdk/v65/functions"

// GetFunctionCredentialMapForTest exports getFunctionCredentialMap for unit testing.
func GetFunctionCredentialMapForTest(fn ocifunctions.Function) map[string][]byte {
	return getFunctionCredentialMap(fn)
}
