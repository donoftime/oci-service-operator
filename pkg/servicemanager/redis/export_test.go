/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package redis

import "github.com/oracle/oci-go-sdk/v65/redis"

// GetCredentialMapForTest exports getCredentialMap for unit testing.
func GetCredentialMapForTest(cluster redis.RedisCluster) map[string][]byte {
	return getCredentialMap(cluster)
}
