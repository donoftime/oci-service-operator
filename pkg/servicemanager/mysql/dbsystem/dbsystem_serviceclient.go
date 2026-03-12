/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dbsystem

import (
	"context"
	"fmt"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/mysql"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/util"
	"reflect"
)

type DbSystemServiceClient interface {
	CreateDbSystem(ctx context.Context, dbSystem ociv1beta1.MySqlDbSystem) (mysql.DbSystem, error)

	UpdateMySqlDbSystem(ctx context.Context, request mysql.UpdateDbSystemRequest) (mysql.UpdateDbSystemResponse, error)

	GetDbSystem(ctx context.Context, request mysql.GetDbSystemRequest) (mysql.GetDbSystemResponse, error)

	DeleteMySqlDbSystem(ctx context.Context, dbSystemId ociv1beta1.OCID) error

	servicemanager.OSOKServiceManager
}

// MySQLDbSystemClientInterface defines the OCI operations used by DbSystemServiceManager.
type MySQLDbSystemClientInterface interface {
	CreateDbSystem(ctx context.Context, request mysql.CreateDbSystemRequest) (mysql.CreateDbSystemResponse, error)
	ListDbSystems(ctx context.Context, request mysql.ListDbSystemsRequest) (mysql.ListDbSystemsResponse, error)
	GetDbSystem(ctx context.Context, request mysql.GetDbSystemRequest) (mysql.GetDbSystemResponse, error)
	UpdateDbSystem(ctx context.Context, request mysql.UpdateDbSystemRequest) (mysql.UpdateDbSystemResponse, error)
	DeleteDbSystem(ctx context.Context, request mysql.DeleteDbSystemRequest) (mysql.DeleteDbSystemResponse, error)
}

func getDbSystemClient(provider common.ConfigurationProvider) (mysql.DbSystemClient, error) {
	return mysql.NewDbSystemClientWithConfigurationProvider(provider)
}

// getOCIClient returns the injected client if set, otherwise creates one from the provider.
func (c *DbSystemServiceManager) getOCIClient() (MySQLDbSystemClientInterface, error) {
	if c.ociClient != nil {
		return c.ociClient, nil
	}
	return getDbSystemClient(c.Provider)
}

func (c *DbSystemServiceManager) CreateDbSystem(ctx context.Context, dbSystem ociv1beta1.MySqlDbSystem, adminUname string, adminPwd string) (mysql.CreateDbSystemResponse, error) {
	dbSystemClient, err := c.getOCIClient()
	if err != nil {
		return mysql.CreateDbSystemResponse{}, err
	}

	c.Log.DebugLog("Creating MySqlDbSystem", "name", dbSystem.Spec.DisplayName)

	createDbSystemDetails := mysql.CreateDbSystemDetails{
		ShapeName:            common.String(dbSystem.Spec.ShapeName),
		AvailabilityDomain:   common.String(dbSystem.Spec.AvailabilityDomain),
		FaultDomain:          common.String(dbSystem.Spec.FaultDomain),
		IsHighlyAvailable:    common.Bool(dbSystem.Spec.IsHighlyAvailable),
		CompartmentId:        common.String(string(dbSystem.Spec.CompartmentId)),
		DataStorageSizeInGBs: common.Int(dbSystem.Spec.DataStorageSizeInGBs),
		SubnetId:             common.String(string(dbSystem.Spec.SubnetId)),
		AdminUsername:        common.String(adminUname),
		AdminPassword:        common.String(adminPwd),
		DisplayName:          common.String(dbSystem.Spec.DisplayName),
		DefinedTags:          *util.ConvertToOciDefinedTags(&dbSystem.Spec.DefinedTags),
		FreeformTags:         dbSystem.Spec.FreeFormTags,
	}

	if dbSystem.Spec.Description != "" {
		createDbSystemDetails.Description = common.String(dbSystem.Spec.Description)
	}

	if dbSystem.Spec.Port != 0 {
		createDbSystemDetails.Port = common.Int(dbSystem.Spec.Port)
	}

	if dbSystem.Spec.PortX != 0 {
		createDbSystemDetails.PortX = common.Int(dbSystem.Spec.PortX)
	}

	if dbSystem.Spec.ConfigurationId.Id != "" {
		createDbSystemDetails.ConfigurationId = common.String(string(dbSystem.Spec.ConfigurationId.Id))
	}

	if dbSystem.Spec.IpAddress != "" {
		createDbSystemDetails.IpAddress = common.String(dbSystem.Spec.IpAddress)
	}

	if dbSystem.Spec.HostnameLabel != "" {
		createDbSystemDetails.HostnameLabel = common.String(dbSystem.Spec.HostnameLabel)
	}

	if dbSystem.Spec.MysqlVersion != "" {
		createDbSystemDetails.MysqlVersion = common.String(dbSystem.Spec.MysqlVersion)
	}

	createDbSystemRequest := mysql.CreateDbSystemRequest{
		CreateDbSystemDetails: createDbSystemDetails,
	}

	return dbSystemClient.CreateDbSystem(ctx, createDbSystemRequest)

}

func (c *DbSystemServiceManager) GetMySqlDbSystemOcid(ctx context.Context, dbSystem ociv1beta1.MySqlDbSystem) (*ociv1beta1.OCID, error) {
	dbSystemClient, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	listDbSystemRequest := mysql.ListDbSystemsRequest{
		CompartmentId: common.String(string(dbSystem.Spec.CompartmentId)),
		DisplayName:   common.String(dbSystem.Spec.DisplayName),
		Limit:         common.Int(1),
	}

	listDbSystemResponse, err := dbSystemClient.ListDbSystems(ctx, listDbSystemRequest)
	if err != nil {
		c.Log.ErrorLog(err, "Error while listing Mysql DB Systems")
		return nil, err
	}

	if len(listDbSystemResponse.Items) > 0 {
		status := listDbSystemResponse.Items[0].LifecycleState

		if status == "ACTIVE" || status == "CREATING" || status == "UPDATING" || status == "INACTIVE" {

			c.Log.DebugLog(fmt.Sprintf("MySql DbSystem %s exists.", dbSystem.Spec.DisplayName))

			return (*ociv1beta1.OCID)(listDbSystemResponse.Items[0].Id), nil
		}
	}
	c.Log.DebugLog(fmt.Sprintf("MySql DbSystem %s does not exist.", dbSystem.Spec.DisplayName))
	return nil, nil
	//
	//c.Log.InfoLog(fmt.Sprintf("Mysql Status ocid %s", dbSystem.Status.OsokStatus.Ocid))
	//
	//// TODO: Implement get mysqldbsystem with ocid populated in status
	//if dbSystem.Status.OsokStatus.Ocid != "" {
	//	dbSystemId := dbSystem.Status.OsokStatus.Ocid
	//
	//	getDbSystemRequest := mysql.GetDbSystemRequest{
	//		DbSystemId: common.String(string(dbSystemId)),
	//	}
	//
	//	getDbsystem, err := dbSystemClient.GetDbSystem(ctx, getDbSystemRequest)
	//	if err != nil {
	//		c.Log.ErrorLog(err, "Error while getting MysqlDb Systems")
	//		return nil, err
	//	}
	//
	//	status := getDbsystem.LifecycleState
	//	if status == "ACTIVE" || status == "CREATING" || status == "UPDATING" || status == "INACTIVE" {
	//		c.Log.DebugLog(fmt.Sprintf("MySql DbSystem %s exists.", dbSystem.Spec.DisplayName))
	//
	//		return (*ociv1beta1.OCID)(getDbsystem.Id), nil
	//	}
	//}
	//c.Log.DebugLog(fmt.Sprintf("MySql DbSystem %s does not exist.", dbSystem.Spec.DisplayName))
	//return nil, nil
}

func (c *DbSystemServiceManager) DeleteMySqlDbSystem(ctx context.Context, dbSystemId ociv1beta1.OCID) error {
	dbClient, err := c.getOCIClient()
	if err != nil {
		return err
	}

	req := mysql.DeleteDbSystemRequest{
		DbSystemId: common.String(string(dbSystemId)),
	}

	_, err = dbClient.DeleteDbSystem(ctx, req)
	return err
}

// GetMySqlDbSystem Sync the MySqlDbSystem details
func (c *DbSystemServiceManager) GetMySqlDbSystem(ctx context.Context, dbSystemId ociv1beta1.OCID, retryPolicy *common.RetryPolicy) (*mysql.DbSystem, error) {
	dbClient, err := c.getOCIClient()
	if err != nil {
		return nil, err
	}

	getDbSystemRequest := mysql.GetDbSystemRequest{
		DbSystemId: common.String(string(dbSystemId)),
	}

	if retryPolicy != nil {
		getDbSystemRequest.RequestMetadata.RetryPolicy = retryPolicy
	}

	response, err := dbClient.GetDbSystem(ctx, getDbSystemRequest)
	if err != nil {
		return nil, err
	}

	return &response.DbSystem, nil
}

func (c *DbSystemServiceManager) UpdateMySqlDbSystem(ctx context.Context, dbSystem *ociv1beta1.MySqlDbSystem) error {
	dbClient, err := c.getOCIClient()
	if err != nil {
		return err
	}

	existingDbSystem, err := c.GetMySqlDbSystem(ctx, dbSystem.Spec.MySqlDbSystemId, nil)
	if err != nil {
		return err
	}

	updateMySqlDbSystemDetails := mysql.UpdateDbSystemDetails{}
	updateNeeded := applyMySQLDisplayNameUpdate(&updateMySqlDbSystemDetails, dbSystem, existingDbSystem)
	updateNeeded = applyMySQLDescriptionUpdate(&updateMySqlDbSystemDetails, dbSystem, existingDbSystem) || updateNeeded
	updateNeeded = applyMySQLConfigurationUpdate(&updateMySqlDbSystemDetails, dbSystem, existingDbSystem) || updateNeeded
	updateNeeded = applyMySQLFreeformTagUpdate(&updateMySqlDbSystemDetails, dbSystem, existingDbSystem) || updateNeeded
	updateNeeded = applyMySQLDefinedTagUpdate(&updateMySqlDbSystemDetails, dbSystem, existingDbSystem) || updateNeeded

	if updateNeeded {
		updateMySqlDbSystemRequest := mysql.UpdateDbSystemRequest{
			DbSystemId:            common.String(string(dbSystem.Spec.MySqlDbSystemId)),
			UpdateDbSystemDetails: updateMySqlDbSystemDetails,
		}

		if _, err := dbClient.UpdateDbSystem(ctx, updateMySqlDbSystemRequest); err != nil {
			return err
		}
	}

	return nil
}

func applyMySQLDisplayNameUpdate(updateDetails *mysql.UpdateDbSystemDetails,
	dbSystem *ociv1beta1.MySqlDbSystem, existingDbSystem *mysql.DbSystem) bool {
	if dbSystem.Spec.DisplayName == "" || *existingDbSystem.DisplayName == dbSystem.Spec.DisplayName {
		return false
	}

	updateDetails.DisplayName = common.String(dbSystem.Spec.DisplayName)
	return true
}

func applyMySQLDescriptionUpdate(updateDetails *mysql.UpdateDbSystemDetails,
	dbSystem *ociv1beta1.MySqlDbSystem, existingDbSystem *mysql.DbSystem) bool {
	if dbSystem.Spec.Description == "" || dbSystem.Spec.Description == *existingDbSystem.Description {
		return false
	}

	updateDetails.Description = common.String(dbSystem.Spec.Description)
	return true
}

func applyMySQLConfigurationUpdate(updateDetails *mysql.UpdateDbSystemDetails,
	dbSystem *ociv1beta1.MySqlDbSystem, existingDbSystem *mysql.DbSystem) bool {
	if dbSystem.Spec.ConfigurationId.Id == "" || string(dbSystem.Spec.ConfigurationId.Id) == *existingDbSystem.ConfigurationId {
		return false
	}

	updateDetails.ConfigurationId = common.String(string(dbSystem.Spec.ConfigurationId.Id))
	return true
}

func applyMySQLFreeformTagUpdate(updateDetails *mysql.UpdateDbSystemDetails,
	dbSystem *ociv1beta1.MySqlDbSystem, existingDbSystem *mysql.DbSystem) bool {
	if dbSystem.Spec.FreeFormTags == nil || reflect.DeepEqual(existingDbSystem.FreeformTags, dbSystem.Spec.FreeFormTags) {
		return false
	}

	updateDetails.FreeformTags = dbSystem.Spec.FreeFormTags
	return true
}

func applyMySQLDefinedTagUpdate(updateDetails *mysql.UpdateDbSystemDetails,
	dbSystem *ociv1beta1.MySqlDbSystem, existingDbSystem *mysql.DbSystem) bool {
	if dbSystem.Spec.DefinedTags == nil {
		return false
	}

	defTag := *util.ConvertToOciDefinedTags(&dbSystem.Spec.DefinedTags)
	if reflect.DeepEqual(existingDbSystem.DefinedTags, defTag) {
		return false
	}

	updateDetails.DefinedTags = defTag
	return true
}
