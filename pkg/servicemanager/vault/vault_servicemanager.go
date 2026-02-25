/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package vault

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/keymanagement"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/oracle/oci-service-operator/pkg/util"
)

// Compile-time check that OciVaultServiceManager implements OSOKServiceManager.
var _ servicemanager.OSOKServiceManager = &OciVaultServiceManager{}

// OciVaultServiceManager implements OSOKServiceManager for OCI Vault (Key Management).
type OciVaultServiceManager struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
}

// NewOciVaultServiceManager creates a new OciVaultServiceManager.
func NewOciVaultServiceManager(provider common.ConfigurationProvider, credClient credhelper.CredentialClient,
	scheme *runtime.Scheme, log loggerutil.OSOKLogger) *OciVaultServiceManager {
	return &OciVaultServiceManager{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           scheme,
		Log:              log,
	}
}

// CreateOrUpdate reconciles the OciVault resource against OCI.
//
// Vault creation is synchronous — the OCI KMS API returns the vault immediately in CREATING
// state and transitions to ACTIVE once provisioning completes. We therefore poll until active.
func (c *OciVaultServiceManager) CreateOrUpdate(ctx context.Context, obj runtime.Object, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	v, err := c.convert(obj)
	if err != nil {
		c.Log.ErrorLog(err, "Conversion of object failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	var vaultInstance *keymanagement.Vault

	if strings.TrimSpace(string(v.Spec.VaultId)) == "" {
		// No explicit ID — look up by display name or create.
		vaultOcid, err := c.GetVaultOcid(ctx, *v)
		if err != nil {
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if vaultOcid == nil {
			// Create new vault.
			vaultInstance, err = c.CreateVault(ctx, *v)
			if err != nil {
				v.Status.OsokStatus = util.UpdateOSOKStatusCondition(v.Status.OsokStatus,
					ociv1beta1.Failed, v1.ConditionFalse, "", err.Error(), c.Log)
				c.Log.ErrorLog(err, "Create OciVault failed")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
		} else {
			vaultInstance, err = c.GetVault(ctx, *vaultOcid)
			if err != nil {
				c.Log.ErrorLog(err, "Error while getting OciVault by OCID")
				return servicemanager.OSOKResponse{IsSuccessful: false}, err
			}
		}

		if vaultInstance.LifecycleState == keymanagement.VaultLifecycleStateCreating {
			c.Log.InfoLog(fmt.Sprintf("OciVault %s is still CREATING", safeString(vaultInstance.DisplayName)))
			v.Status.OsokStatus = util.UpdateOSOKStatusCondition(v.Status.OsokStatus,
				ociv1beta1.Provisioning, v1.ConditionTrue, "", "OciVault Provisioning", c.Log)
			v.Status.OsokStatus.Ocid = ociv1beta1.OCID(*vaultInstance.Id)
			return servicemanager.OSOKResponse{IsSuccessful: false}, nil
		}

		v.Status.OsokStatus.Ocid = ociv1beta1.OCID(*vaultInstance.Id)
		v.Status.OsokStatus = util.UpdateOSOKStatusCondition(v.Status.OsokStatus,
			ociv1beta1.Active, v1.ConditionTrue, "",
			fmt.Sprintf("OciVault %s is %s", safeString(vaultInstance.DisplayName), vaultInstance.LifecycleState), c.Log)
		c.Log.InfoLog(fmt.Sprintf("OciVault %s is %s", safeString(vaultInstance.DisplayName), vaultInstance.LifecycleState))

	} else {
		// Bind to an existing vault by ID.
		vaultInstance, err = c.GetVault(ctx, v.Spec.VaultId)
		if err != nil {
			c.Log.ErrorLog(err, "Error while getting existing OciVault")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		if err = c.UpdateVault(ctx, v); err != nil {
			c.Log.ErrorLog(err, "Error while updating OciVault")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}

		v.Status.OsokStatus = util.UpdateOSOKStatusCondition(v.Status.OsokStatus,
			ociv1beta1.Active, v1.ConditionTrue, "", "OciVault Bound/Updated", c.Log)
		c.Log.InfoLog(fmt.Sprintf("OciVault %s is bound/updated", safeString(vaultInstance.DisplayName)))
	}

	v.Status.OsokStatus.Ocid = ociv1beta1.OCID(*vaultInstance.Id)
	if v.Status.OsokStatus.CreatedAt == nil {
		now := metav1.NewTime(time.Now())
		v.Status.OsokStatus.CreatedAt = &now
	}

	if vaultInstance.LifecycleState == keymanagement.VaultLifecycleStatePendingDeletion ||
		vaultInstance.LifecycleState == keymanagement.VaultLifecycleStateDeleting ||
		vaultInstance.LifecycleState == keymanagement.VaultLifecycleStateDeleted {
		v.Status.OsokStatus = util.UpdateOSOKStatusCondition(v.Status.OsokStatus,
			ociv1beta1.Failed, v1.ConditionFalse, "",
			fmt.Sprintf("OciVault %s is in state %s", safeString(vaultInstance.DisplayName), vaultInstance.LifecycleState), c.Log)
		return servicemanager.OSOKResponse{IsSuccessful: false}, nil
	}

	// Handle optional key creation/binding within the vault.
	if v.Spec.Key != nil && vaultInstance.ManagementEndpoint != nil {
		if err := c.reconcileKey(ctx, v, *vaultInstance.ManagementEndpoint); err != nil {
			c.Log.ErrorLog(err, "Error reconciling key within OciVault")
			return servicemanager.OSOKResponse{IsSuccessful: false}, err
		}
	}

	_, err = c.addToSecret(ctx, v.Namespace, v.Name, *vaultInstance)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return servicemanager.OSOKResponse{IsSuccessful: true}, nil
		}
		c.Log.InfoLog("Secret creation failed")
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	return servicemanager.OSOKResponse{IsSuccessful: true}, nil
}

// reconcileKey ensures the key specified in the spec exists within the vault.
func (c *OciVaultServiceManager) reconcileKey(ctx context.Context, v *ociv1beta1.OciVault, managementEndpoint string) error {
	if strings.TrimSpace(string(v.Spec.Key.KeyId)) != "" {
		// Bind to existing key — just verify it exists.
		_, err := c.GetKey(ctx, v.Spec.Key.KeyId, managementEndpoint)
		return err
	}

	// Look up by display name or create.
	keyOcid, err := c.GetKeyOcid(ctx, *v, managementEndpoint)
	if err != nil {
		return err
	}

	if keyOcid == nil {
		key, err := c.CreateKey(ctx, *v, managementEndpoint)
		if err != nil {
			return fmt.Errorf("creating key in vault: %w", err)
		}
		c.Log.InfoLog(fmt.Sprintf("Created key %s in vault %s", *key.Id, v.Status.OsokStatus.Ocid))
	} else {
		c.Log.InfoLog(fmt.Sprintf("Key %s already exists in vault", *keyOcid))
	}
	return nil
}

// Delete handles deletion of the Vault (called by the finalizer).
func (c *OciVaultServiceManager) Delete(ctx context.Context, obj runtime.Object) (bool, error) {
	v, err := c.convert(obj)
	if err != nil {
		return false, err
	}

	if v.Status.OsokStatus.Ocid == "" {
		c.Log.InfoLog("OciVault has no OCID, nothing to delete")
		return true, nil
	}

	// Retrieve the vault to get the management endpoint for key deletion.
	vaultInstance, err := c.GetVault(ctx, v.Status.OsokStatus.Ocid)
	if err != nil {
		c.Log.ErrorLog(err, "Error getting OciVault for deletion")
		return false, err
	}

	// Schedule key deletion first if a key was managed.
	if v.Spec.Key != nil && vaultInstance.ManagementEndpoint != nil {
		var keyOcid *ociv1beta1.OCID
		if strings.TrimSpace(string(v.Spec.Key.KeyId)) != "" {
			keyOcid = &v.Spec.Key.KeyId
		} else {
			keyOcid, err = c.GetKeyOcid(ctx, *v, *vaultInstance.ManagementEndpoint)
			if err != nil {
				c.Log.ErrorLog(err, "Error looking up key for deletion")
			}
		}
		if keyOcid != nil {
			if err := c.ScheduleKeyDeletion(ctx, *keyOcid, *vaultInstance.ManagementEndpoint); err != nil {
				c.Log.ErrorLog(err, "Error scheduling key deletion")
			} else {
				c.Log.InfoLog(fmt.Sprintf("Scheduled deletion of key %s", *keyOcid))
			}
		}
	}

	c.Log.InfoLog(fmt.Sprintf("Scheduling deletion of OciVault %s", v.Status.OsokStatus.Ocid))
	if err := c.ScheduleVaultDeletion(ctx, v.Status.OsokStatus.Ocid); err != nil {
		c.Log.ErrorLog(err, "Error while scheduling OciVault deletion")
		return false, err
	}

	if _, err := c.CredentialClient.DeleteSecret(ctx, v.Name, v.Namespace); err != nil {
		c.Log.ErrorLog(err, "Error while deleting OciVault secret")
	}

	return true, nil
}

// GetCrdStatus returns the OSOK status from the resource.
func (c *OciVaultServiceManager) GetCrdStatus(obj runtime.Object) (*ociv1beta1.OSOKStatus, error) {
	resource, err := c.convert(obj)
	if err != nil {
		return nil, err
	}
	return &resource.Status.OsokStatus, nil
}

func (c *OciVaultServiceManager) convert(obj runtime.Object) (*ociv1beta1.OciVault, error) {
	v, ok := obj.(*ociv1beta1.OciVault)
	if !ok {
		return nil, fmt.Errorf("failed type assertion for OciVault")
	}
	return v, nil
}

func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
