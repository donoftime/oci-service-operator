/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package objectstorage

import (
	"context"
	"fmt"
)

func (m *ObjectStorageBucketServiceManager) addToSecret(ctx context.Context, k8sNamespace string, resourceName string,
	namespace string, bucketName string) (bool, error) {

	m.Log.InfoLog("Creating the ObjectStorageBucket connection secret")
	credMap := getCredentialMap(namespace, bucketName)

	m.Log.InfoLog(fmt.Sprintf("Creating secret for ObjectStorageBucket %s in namespace %s", resourceName, k8sNamespace))
	return m.CredentialClient.CreateSecret(ctx, resourceName, k8sNamespace, nil, credMap)
}

func getCredentialMap(namespace, bucketName string) map[string][]byte {
	endpoint := fmt.Sprintf("https://objectstorage.<region>.oraclecloud.com/n/%s/b/%s", namespace, bucketName)
	return map[string][]byte{
		"namespace":   []byte(namespace),
		"bucketName":  []byte(bucketName),
		"apiEndpoint": []byte(endpoint),
	}
}
