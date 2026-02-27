/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package config

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/stretchr/testify/assert"
)

func testLogger() loggerutil.OSOKLogger {
	return loggerutil.OSOKLogger{Logger: logr.Discard()}
}

// ---------------------------------------------------------------------------
// Tests: osokConfig methods (Auth, UseInstancePrincipals, VaultDetails)
// ---------------------------------------------------------------------------

func TestOsokConfig_Auth(t *testing.T) {
	cfg := osokConfig{
		auth: UserAuthConfig{Tenancy: "t1", User: "u1", Region: "us-phoenix-1"},
	}
	auth := cfg.Auth()
	assert.Equal(t, "t1", auth.Tenancy)
	assert.Equal(t, "u1", auth.User)
	assert.Equal(t, "us-phoenix-1", auth.Region)
}

func TestOsokConfig_UseInstancePrincipals_True(t *testing.T) {
	cfg := osokConfig{useInstancePrincipals: true}
	assert.True(t, cfg.UseInstancePrincipals())
}

func TestOsokConfig_UseInstancePrincipals_False(t *testing.T) {
	cfg := osokConfig{useInstancePrincipals: false}
	assert.False(t, cfg.UseInstancePrincipals())
}

func TestOsokConfig_VaultDetails(t *testing.T) {
	cfg := osokConfig{vaultDetails: "ocid1.vault.oc1..xxx"}
	assert.Equal(t, "ocid1.vault.oc1..xxx", cfg.VaultDetails())
}

func TestOsokConfig_ImplementsInterface(t *testing.T) {
	var _ OsokConfig = osokConfig{}
}

// ---------------------------------------------------------------------------
// Tests: GetConfigDetails — env-driven
// ---------------------------------------------------------------------------

func TestGetConfigDetails_InstancePrincipalTrue(t *testing.T) {
	t.Setenv("USEINSTANCEPRINCIPAL", "true")
	t.Setenv("VAULTDETAILS", "")
	t.Setenv("USER", "")
	t.Setenv("TENANCY", "")
	t.Setenv("REGION", "")
	t.Setenv("FINGERPRINT", "")
	t.Setenv("PASSPHRASE", "")
	t.Setenv("PRIVATEKEY", "")

	// Reset global state before test.
	configDetails = osokConfig{}
	cfg := GetConfigDetails(testLogger())
	assert.True(t, cfg.UseInstancePrincipals())
}

func TestGetConfigDetails_InstancePrincipalFalse(t *testing.T) {
	t.Setenv("USEINSTANCEPRINCIPAL", "false")
	t.Setenv("VAULTDETAILS", "")
	t.Setenv("USER", "")
	t.Setenv("TENANCY", "")
	t.Setenv("REGION", "")
	t.Setenv("FINGERPRINT", "")
	t.Setenv("PASSPHRASE", "")
	t.Setenv("PRIVATEKEY", "")

	configDetails = osokConfig{}
	cfg := GetConfigDetails(testLogger())
	assert.False(t, cfg.UseInstancePrincipals())
}

func TestGetConfigDetails_InstancePrincipalInvalidValue(t *testing.T) {
	t.Setenv("USEINSTANCEPRINCIPAL", "not-a-bool")
	t.Setenv("VAULTDETAILS", "")
	t.Setenv("USER", "")
	t.Setenv("TENANCY", "")
	t.Setenv("REGION", "")
	t.Setenv("FINGERPRINT", "")
	t.Setenv("PASSPHRASE", "")
	t.Setenv("PRIVATEKEY", "")

	configDetails = osokConfig{}
	cfg := GetConfigDetails(testLogger())
	// ParseBool fails, so useInstancePrincipals set to false first, then val=false.
	// Either way it should not panic and should return a valid config.
	assert.NotNil(t, cfg)
}

func TestGetConfigDetails_NoEnvVars(t *testing.T) {
	t.Setenv("USEINSTANCEPRINCIPAL", "")
	t.Setenv("VAULTDETAILS", "")
	t.Setenv("USER", "")
	t.Setenv("TENANCY", "")
	t.Setenv("REGION", "")
	t.Setenv("FINGERPRINT", "")
	t.Setenv("PASSPHRASE", "")
	t.Setenv("PRIVATEKEY", "")

	configDetails = osokConfig{}
	cfg := GetConfigDetails(testLogger())
	assert.False(t, cfg.UseInstancePrincipals())
	assert.Equal(t, "", cfg.VaultDetails())
}

func TestGetConfigDetails_VaultDetails(t *testing.T) {
	t.Setenv("USEINSTANCEPRINCIPAL", "")
	t.Setenv("VAULTDETAILS", "ocid1.vault.oc1..testvault")
	t.Setenv("USER", "")
	t.Setenv("TENANCY", "")
	t.Setenv("REGION", "")
	t.Setenv("FINGERPRINT", "")
	t.Setenv("PASSPHRASE", "")
	t.Setenv("PRIVATEKEY", "")

	configDetails = osokConfig{}
	cfg := GetConfigDetails(testLogger())
	assert.Equal(t, "ocid1.vault.oc1..testvault", cfg.VaultDetails())
}

// ---------------------------------------------------------------------------
// Tests: SetUserConfigDetails — env-driven
// ---------------------------------------------------------------------------

func TestSetUserConfigDetails_AllFields(t *testing.T) {
	t.Setenv("USER", "testuser")
	t.Setenv("TENANCY", "testtenancy")
	t.Setenv("REGION", "us-ashburn-1")
	t.Setenv("FINGERPRINT", "aa:bb:cc")
	t.Setenv("PASSPHRASE", "testpass")
	t.Setenv("PRIVATEKEY", "-----BEGIN RSA PRIVATE KEY-----")

	configDetails = osokConfig{}
	SetUserConfigDetails(testLogger())

	assert.Equal(t, "testuser", configDetails.auth.User)
	assert.Equal(t, "testtenancy", configDetails.auth.Tenancy)
	assert.Equal(t, "us-ashburn-1", configDetails.auth.Region)
	assert.Equal(t, "aa:bb:cc", configDetails.auth.Fingerprint)
	assert.Equal(t, "testpass", configDetails.auth.Passphrase)
	assert.Equal(t, "-----BEGIN RSA PRIVATE KEY-----", configDetails.auth.PrivateKey)
}

func TestSetUserConfigDetails_NoEnvVars(t *testing.T) {
	t.Setenv("USER", "")
	t.Setenv("TENANCY", "")
	t.Setenv("REGION", "")
	t.Setenv("FINGERPRINT", "")
	t.Setenv("PASSPHRASE", "")
	t.Setenv("PRIVATEKEY", "")

	configDetails = osokConfig{}
	SetUserConfigDetails(testLogger())

	assert.Equal(t, "", configDetails.auth.User)
	assert.Equal(t, "", configDetails.auth.Tenancy)
	assert.Equal(t, "", configDetails.auth.Region)
}

func TestSetUserConfigDetails_PartialFields(t *testing.T) {
	t.Setenv("USER", "partial-user")
	t.Setenv("TENANCY", "")
	t.Setenv("REGION", "eu-frankfurt-1")
	t.Setenv("FINGERPRINT", "")
	t.Setenv("PASSPHRASE", "")
	t.Setenv("PRIVATEKEY", "")

	configDetails = osokConfig{}
	SetUserConfigDetails(testLogger())

	assert.Equal(t, "partial-user", configDetails.auth.User)
	assert.Equal(t, "", configDetails.auth.Tenancy)
	assert.Equal(t, "eu-frankfurt-1", configDetails.auth.Region)
}
