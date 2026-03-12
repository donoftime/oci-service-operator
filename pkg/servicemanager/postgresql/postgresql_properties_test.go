/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package postgresql_test

import (
	"context"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocipsql "github.com/oracle/oci-go-sdk/v65/psql"
	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

func makePendingPostgresDbSystem(id, name string, state ocipsql.DbSystemLifecycleStateEnum) ocipsql.DbSystem {
	db := makeActiveDbSystem(id, name)
	db.LifecycleState = state
	return db
}

func makePostgresSpec(name string) *ociv1beta1.PostgresDbSystem {
	db := &ociv1beta1.PostgresDbSystem{}
	db.Name = name
	db.Namespace = "default"
	db.Spec.DisplayName = name
	db.Spec.CompartmentId = "ocid1.compartment.oc1..x"
	db.Spec.DbVersion = "14.10"
	db.Spec.Shape = "VM.Standard.E4.Flex"
	db.Spec.SubnetId = "ocid1.subnet.oc1..x"
	return db
}

func TestPropertyPostgresPendingStatesRequestRequeue(t *testing.T) {
	for _, state := range []ocipsql.DbSystemLifecycleStateEnum{
		ocipsql.DbSystemLifecycleStateCreating,
		ocipsql.DbSystemLifecycleStateUpdating,
	} {
		t.Run(string(state), func(t *testing.T) {
			ociClient := &mockOciPostgresClient{
				listFn: func(_ context.Context, _ ocipsql.ListDbSystemsRequest) (ocipsql.ListDbSystemsResponse, error) {
					return ocipsql.ListDbSystemsResponse{
						DbSystemCollection: ocipsql.DbSystemCollection{
							Items: []ocipsql.DbSystemSummary{{Id: common.String("ocid1.postgresql.oc1..pending"), DisplayName: common.String("pending-db"), LifecycleState: state}},
						},
					}, nil
				},
				getFn: func(_ context.Context, _ ocipsql.GetDbSystemRequest) (ocipsql.GetDbSystemResponse, error) {
					return ocipsql.GetDbSystemResponse{DbSystem: makePendingPostgresDbSystem("ocid1.postgresql.oc1..pending", "pending-db", state)}, nil
				},
			}
			mgr := newPostgresMgr(t, ociClient, &fakeCredentialClient{})
			db := makePostgresSpec("pending-db")

			resp, err := mgr.CreateOrUpdate(context.Background(), db, ctrl.Request{})
			assert.NoError(t, err)
			assert.False(t, resp.IsSuccessful)
			assert.True(t, resp.ShouldRequeue)
		})
	}
}

func TestPropertyPostgresBindByIDUsesSpecIDWhenStatusIsEmpty(t *testing.T) {
	var updatedID string
	ociClient := &mockOciPostgresClient{
		getFn: func(_ context.Context, req ocipsql.GetDbSystemRequest) (ocipsql.GetDbSystemResponse, error) {
			return ocipsql.GetDbSystemResponse{DbSystem: makeActiveDbSystem(*req.DbSystemId, "old-bound-db")}, nil
		},
		updateFn: func(_ context.Context, req ocipsql.UpdateDbSystemRequest) (ocipsql.UpdateDbSystemResponse, error) {
			updatedID = *req.DbSystemId
			return ocipsql.UpdateDbSystemResponse{}, nil
		},
	}
	mgr := newPostgresMgr(t, ociClient, &fakeCredentialClient{})
	db := makePostgresSpec("new-bound-db")
	db.Spec.PostgresDbSystemId = "ocid1.postgresql.oc1..bind"

	resp, err := mgr.CreateOrUpdate(context.Background(), db, ctrl.Request{})
	assert.NoError(t, err)
	assert.True(t, resp.IsSuccessful)
	assert.Equal(t, string(db.Spec.PostgresDbSystemId), updatedID)
}

func TestPropertyPostgresDeleteWaitsForConfirmedDisappearance(t *testing.T) {
	ociClient := &mockOciPostgresClient{
		deleteFn: func(_ context.Context, _ ocipsql.DeleteDbSystemRequest) (ocipsql.DeleteDbSystemResponse, error) {
			return ocipsql.DeleteDbSystemResponse{}, nil
		},
		getFn: func(_ context.Context, req ocipsql.GetDbSystemRequest) (ocipsql.GetDbSystemResponse, error) {
			return ocipsql.GetDbSystemResponse{DbSystem: makeActiveDbSystem(*req.DbSystemId, "still-there")}, nil
		},
	}
	credClient := &fakeCredentialClient{}
	mgr := newPostgresMgr(t, ociClient, credClient)
	db := makePostgresSpec("still-there")
	db.Status.OsokStatus.Ocid = "ocid1.postgresql.oc1..delete"

	done, err := mgr.Delete(context.Background(), db)
	assert.NoError(t, err)
	assert.False(t, done)
	assert.False(t, credClient.deleteCalled)
}
