/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package containerinstance

import (
	"context"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/containerinstances"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
)

// ExportSetClientForTest sets the OCI client on the service manager for unit testing.
func ExportSetClientForTest(m *ContainerInstanceServiceManager, c ContainerInstanceClientInterface) {
	m.ociClient = c
}

// GetRetryPolicyForTest exports getRetryPolicy for unit testing.
func GetRetryPolicyForTest(c *ContainerInstanceServiceManager, attempts uint) common.RetryPolicy {
	return c.getRetryPolicy(attempts)
}

// ExportListAllContainerInstances exports ListAllContainerInstances for unit testing.
func ExportListAllContainerInstances(
	m *ContainerInstanceServiceManager,
	ctx context.Context,
	ci ociv1beta1.ContainerInstance,
) ([]containerinstances.ContainerInstanceSummary, error) {
	return m.ListAllContainerInstances(ctx, ci)
}

// ExportGarbageCollect exports GarbageCollect for unit testing.
func ExportGarbageCollect(
	m *ContainerInstanceServiceManager,
	ctx context.Context,
	ci ociv1beta1.ContainerInstance,
) error {
	return m.GarbageCollect(ctx, ci)
}
