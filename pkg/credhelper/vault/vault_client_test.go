/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package vault

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/stretchr/testify/assert"
)

// generateTestPEM generates a throwaway RSA private key in PEM format for unit tests.
func generateTestPEM(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	return string(pem.EncodeToMemory(block))
}

func makeTestClient() *VaultClient {
	return NewVaultClient(
		common.NewRawConfigurationProvider("tenancy", "user", "us-ashburn-1", "fingerprint", "privatekey", nil),
		logr.Discard(),
		"ocid1.key.oc1..test",
		"ocid1.vault.oc1..test",
	)
}

func TestNewVaultClient_FieldsSet(t *testing.T) {
	provider := common.NewRawConfigurationProvider("tenancy", "user", "us-ashburn-1", "fp", "pk", nil)
	log := logr.Discard()
	keyId := "ocid1.key.oc1..abc"
	vaultId := "ocid1.vault.oc1..xyz"

	vc := NewVaultClient(provider, log, keyId, vaultId)
	assert.NotNil(t, vc)
	assert.Equal(t, keyId, vc.KeyId)
	assert.Equal(t, vaultId, vc.VaultId)
}

func TestDeleteSecret_ReturnsTrue(t *testing.T) {
	vc := makeTestClient()
	ctx := context.Background()

	ok, err := vc.DeleteSecret(ctx, "my-secret", "default")
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestGetSecret_ReturnsNil(t *testing.T) {
	vc := makeTestClient()
	ctx := context.Background()

	data, err := vc.GetSecret(ctx, "my-secret", "default")
	assert.NoError(t, err)
	assert.Nil(t, data)
}

func TestCreateSecret_FailsWithInvalidProvider(t *testing.T) {
	vc := makeTestClient()
	ctx := context.Background()

	labels := map[string]string{"env": "test"}
	secretData := map[string][]byte{"key": []byte("value")}

	// CreateSecret calls the OCI API which will fail with invalid credentials.
	ok, err := vc.CreateSecret(ctx, "test-secret", "default", labels, secretData)
	assert.Error(t, err, "expected error from OCI API with invalid credentials")
	assert.False(t, ok)
}

func TestCreateSecret_EmptyData(t *testing.T) {
	vc := makeTestClient()
	ctx := context.Background()

	labels := map[string]string{}
	secretData := map[string][]byte{}

	// Empty data still hits OCI API and fails with invalid creds.
	ok, err := vc.CreateSecret(ctx, "empty-secret", "default", labels, secretData)
	assert.Error(t, err)
	assert.False(t, ok)
}

func TestCreateSecret_ValidClientFailsOnAPICall(t *testing.T) {
	// Use a properly-formatted RSA key so client creation succeeds,
	// then the OCI API call will fail (no real endpoint).
	pemKey := generateTestPEM(t)
	provider := common.NewRawConfigurationProvider(
		"ocid1.tenancy.oc1..fake",
		"ocid1.user.oc1..fake",
		"us-ashburn-1",
		"aa:bb:cc:dd:ee:ff:00:11:22:33:44:55:66:77:88:99",
		pemKey,
		nil,
	)
	vc := NewVaultClient(provider, logr.Discard(), "ocid1.key.oc1..test", "ocid1.vault.oc1..test")
	ctx := context.Background()

	labels := map[string]string{"env": "test"}
	secretData := map[string][]byte{"token": []byte("secret-value")}

	// Client creation succeeds; the OCI API call itself will fail.
	ok, err := vc.CreateSecret(ctx, "test-secret", "default", labels, secretData)
	assert.Error(t, err)
	assert.False(t, ok)
}
