/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package controllers

import (
	"context"
	"github.com/oracle/oci-service-operator/pkg/core"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// ObjectStorageBucketReconciler reconciles an ObjectStorageBucket object
type ObjectStorageBucketReconciler struct {
	Reconciler *core.BaseReconciler
}

// +kubebuilder:rbac:groups=oci.oracle.com,resources=objectstoragebuckets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=oci.oracle.com,resources=objectstoragebuckets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=oci.oracle.com,resources=objectstoragebuckets/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ObjectStorageBucketReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	bucket := &ociv1beta1.ObjectStorageBucket{}
	return r.Reconciler.Reconcile(ctx, req, bucket)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ObjectStorageBucketReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ociv1beta1.ObjectStorageBucket{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 3}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
