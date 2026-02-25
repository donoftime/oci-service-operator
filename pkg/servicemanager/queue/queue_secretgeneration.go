/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package queue

import (
	"context"
	"fmt"

	ociqueue "github.com/oracle/oci-go-sdk/v65/queue"
)

func (c *OciQueueServiceManager) addToSecret(ctx context.Context, namespace string, queueName string,
	q ociqueue.Queue) (bool, error) {

	c.Log.InfoLog("Creating the OciQueue connection secret")
	credMap := getCredentialMap(q)

	c.Log.InfoLog(fmt.Sprintf("Creating secret for OciQueue %s in namespace %s", queueName, namespace))
	return c.CredentialClient.CreateSecret(ctx, queueName, namespace, nil, credMap)
}

func getCredentialMap(q ociqueue.Queue) map[string][]byte {
	credMap := make(map[string][]byte)

	if q.Id != nil {
		credMap["id"] = []byte(*q.Id)
	}
	if q.MessagesEndpoint != nil {
		credMap["messagesEndpoint"] = []byte(*q.MessagesEndpoint)
	}
	if q.DisplayName != nil {
		credMap["displayName"] = []byte(*q.DisplayName)
	}

	return credMap
}
