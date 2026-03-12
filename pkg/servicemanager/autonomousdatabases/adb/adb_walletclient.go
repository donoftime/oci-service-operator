/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package adb

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/database"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func (c *AdbServiceManager) GenerateWallet(ctx context.Context, adbId string, adbDisplayName string,
	walletSecretName string, namespace string, walletName string, adbInstanceName string) (bool, error) {
	walletName = resolveWalletName(walletName, adbInstanceName, c.Log)
	exists, err := c.walletSecretExists(ctx, walletName, namespace, adbInstanceName)
	if exists || err != nil {
		return exists, err
	}

	pwd, err := c.getWalletPassword(ctx, walletSecretName, namespace)
	if err != nil {
		return false, err
	}

	dbClient, err := getDbClient(c.Provider)
	if err != nil {
		return false, err
	}

	credMap, err := c.generateWalletCredentials(ctx, dbClient, adbId, adbDisplayName, pwd)
	if err != nil {
		return false, err
	}

	c.Log.InfoLog("Creating the Wallet secret")
	created, err := servicemanager.EnsureOwnedSecret(ctx, c.CredentialClient, walletName, namespace, autonomousDatabaseKindName, adbInstanceName, credMap)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return true, nil
		}
		return false, err
	}
	return created, nil
}

func resolveWalletName(walletName string, adbInstanceName string, log loggerutil.OSOKLogger) string {
	if walletName == "" {
		log.DebugLog("Autonomous Database wallet password name was not provided. Setting default")
		return fmt.Sprintf("%s-wallet", adbInstanceName)
	}
	return walletName
}

func (c *AdbServiceManager) walletSecretExists(ctx context.Context, walletName string, namespace string,
	adbInstanceName string) (bool, error) {
	c.Log.InfoLog("Checking if the wallet secret already exists")
	existingSecret, err := c.CredentialClient.GetSecret(ctx, walletName, namespace)
	if err == nil {
		if servicemanager.SecretOwnedBy(existingSecret, autonomousDatabaseKindName, adbInstanceName) {
			c.Log.InfoLog("Wallet already exists. Not generating wallet.")
			return true, nil
		}
		return false, fmt.Errorf("wallet secret %s/%s already exists and is not owned by autonomous database %s", namespace, walletName, adbInstanceName)
	}
	if !servicemanager.IsSecretNotFoundError(err) {
		return false, err
	}

	return false, nil
}

func (c *AdbServiceManager) generateWalletCredentials(ctx context.Context, dbClient database.DatabaseClient,
	adbId string, adbDisplayName string, pwd *string) (map[string][]byte, error) {
	retryPolicy := c.getExponentialBackoffRetryPolicy(8)
	req := database.GenerateAutonomousDatabaseWalletRequest{
		AutonomousDatabaseId: &adbId,
		GenerateAutonomousDatabaseWalletDetails: database.GenerateAutonomousDatabaseWalletDetails{
			Password: pwd,
		},
		RequestMetadata: common.RequestMetadata{RetryPolicy: &retryPolicy},
	}

	c.Log.InfoLog("Generating the Autonomous Database Wallet")
	resp, err := dbClient.GenerateAutonomousDatabaseWallet(ctx, req)
	if err != nil {
		c.Log.ErrorLog(err, fmt.Sprintf("Error while generating wallet for Autonomous Database %s", adbDisplayName))
		return nil, err
	}

	c.Log.InfoLog("Creating the Credential Map")
	credMap, err := getCredentialMap(adbDisplayName, resp)
	if err != nil {
		c.Log.ErrorLog(err, "Error while creating wallet map")
		return nil, err
	}
	return credMap, nil
}

func getCredentialMap(adbDisplayName string, resp database.GenerateAutonomousDatabaseWalletResponse) (credMap map[string][]byte, err error) {
	tempZip, err := os.CreateTemp("", fmt.Sprintf("%s-wallet*.zip", adbDisplayName))
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := tempZip.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()

	if _, err := io.Copy(tempZip, resp.Content); err != nil {
		return nil, err
	}

	return readWalletZipEntries(tempZip.Name())
}

func readWalletZipEntries(zipPath string) (credMap map[string][]byte, err error) {
	credMap = make(map[string][]byte)

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}

	defer func() {
		if closeErr := reader.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()
	for _, file := range reader.File {
		reader, err := file.Open()
		if err != nil {
			return nil, err
		}

		content, err := io.ReadAll(reader)
		closeErr := reader.Close()
		if err != nil {
			return nil, err
		}
		if closeErr != nil {
			return nil, closeErr
		}
		credMap[file.Name] = content
	}

	return credMap, nil
}

func (c *AdbServiceManager) getWalletPassword(ctx context.Context, secretName string, ns string) (*string, error) {
	secretMap, err := c.CredentialClient.GetSecret(ctx, secretName, ns)
	if err != nil {
		c.Log.ErrorLog(err, "Error while getting the wallet password secret")
		return nil, err
	}

	pwd, ok := secretMap["walletPassword"]
	if !ok {
		c.Log.ErrorLog(err, "password key 'walletPassword' in wallet password secret is not found")
		return nil, errors.New("password key 'walletPassword' in wallet password secret is not found")
	}

	pwdString := string(pwd)
	return &pwdString, nil
}

func (c *AdbServiceManager) getExponentialBackoffRetryPolicy(attempts uint) common.RetryPolicy {
	shouldRetry := func(response common.OCIOperationResponse) bool {
		return response.Error != nil || response.Response.HTTPResponse().StatusCode < 200 ||
			response.Response.HTTPResponse().StatusCode >= 300
	}
	nextDuration := func(response common.OCIOperationResponse) time.Duration {
		return time.Duration(math.Pow(float64(2), float64(response.AttemptNumber-1))) * time.Second
	}
	return common.NewRetryPolicy(attempts, shouldRetry, nextDuration)
}
