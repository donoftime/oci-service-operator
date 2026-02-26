/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package compute

import "github.com/oracle/oci-go-sdk/v65/common"

// ExportSetClientForTest sets the OCI client on the service manager for unit testing.
func ExportSetClientForTest(m *ComputeInstanceServiceManager, c ComputeInstanceClientInterface) {
	m.ociClient = c
}

// GetRetryPolicyForTest exports getRetryPolicy for unit testing.
func GetRetryPolicyForTest(c *ComputeInstanceServiceManager, attempts uint) common.RetryPolicy {
	return c.getRetryPolicy(attempts)
}
