/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package controllers

import (
	"context"

	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/core"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// OciVcnReconciler reconciles an OciVcn object
type OciVcnReconciler struct {
	Reconciler *core.BaseReconciler
}

// +kubebuilder:rbac:groups=oci.oracle.com,resources=ocivcns,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=oci.oracle.com,resources=ocivcns/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=oci.oracle.com,resources=ocivcns/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *OciVcnReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	vcn := &ociv1beta1.OciVcn{}
	return r.Reconciler.Reconcile(ctx, req, vcn)
}

// SetupWithManager sets up the controller with the Manager.
func (r *OciVcnReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ociv1beta1.OciVcn{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 3}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

// OciSubnetReconciler reconciles an OciSubnet object
type OciSubnetReconciler struct {
	Reconciler *core.BaseReconciler
}

// +kubebuilder:rbac:groups=oci.oracle.com,resources=ocisubnets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=oci.oracle.com,resources=ocisubnets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=oci.oracle.com,resources=ocisubnets/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *OciSubnetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	subnet := &ociv1beta1.OciSubnet{}
	return r.Reconciler.Reconcile(ctx, req, subnet)
}

// SetupWithManager sets up the controller with the Manager.
func (r *OciSubnetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ociv1beta1.OciSubnet{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 3}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

// OciInternetGatewayReconciler reconciles an OciInternetGateway object
type OciInternetGatewayReconciler struct {
	Reconciler *core.BaseReconciler
}

// +kubebuilder:rbac:groups=oci.oracle.com,resources=ociinternetgateways,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=oci.oracle.com,resources=ociinternetgateways/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=oci.oracle.com,resources=ociinternetgateways/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *OciInternetGatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	igw := &ociv1beta1.OciInternetGateway{}
	return r.Reconciler.Reconcile(ctx, req, igw)
}

// SetupWithManager sets up the controller with the Manager.
func (r *OciInternetGatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ociv1beta1.OciInternetGateway{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 3}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

// OciNatGatewayReconciler reconciles an OciNatGateway object
type OciNatGatewayReconciler struct {
	Reconciler *core.BaseReconciler
}

// +kubebuilder:rbac:groups=oci.oracle.com,resources=ocinatgateways,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=oci.oracle.com,resources=ocinatgateways/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=oci.oracle.com,resources=ocinatgateways/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *OciNatGatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	nat := &ociv1beta1.OciNatGateway{}
	return r.Reconciler.Reconcile(ctx, req, nat)
}

// SetupWithManager sets up the controller with the Manager.
func (r *OciNatGatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ociv1beta1.OciNatGateway{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 3}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

// OciServiceGatewayReconciler reconciles an OciServiceGateway object
type OciServiceGatewayReconciler struct {
	Reconciler *core.BaseReconciler
}

// +kubebuilder:rbac:groups=oci.oracle.com,resources=ociservicegateways,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=oci.oracle.com,resources=ociservicegateways/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=oci.oracle.com,resources=ociservicegateways/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *OciServiceGatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	sgw := &ociv1beta1.OciServiceGateway{}
	return r.Reconciler.Reconcile(ctx, req, sgw)
}

// SetupWithManager sets up the controller with the Manager.
func (r *OciServiceGatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ociv1beta1.OciServiceGateway{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 3}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

// OciDrgReconciler reconciles an OciDrg object
type OciDrgReconciler struct {
	Reconciler *core.BaseReconciler
}

// +kubebuilder:rbac:groups=oci.oracle.com,resources=ocidrgs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=oci.oracle.com,resources=ocidrgs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=oci.oracle.com,resources=ocidrgs/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *OciDrgReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	drg := &ociv1beta1.OciDrg{}
	return r.Reconciler.Reconcile(ctx, req, drg)
}

// SetupWithManager sets up the controller with the Manager.
func (r *OciDrgReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ociv1beta1.OciDrg{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 3}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

// OciSecurityListReconciler reconciles an OciSecurityList object
type OciSecurityListReconciler struct {
	Reconciler *core.BaseReconciler
}

// +kubebuilder:rbac:groups=oci.oracle.com,resources=ocisecuritylists,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=oci.oracle.com,resources=ocisecuritylists/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=oci.oracle.com,resources=ocisecuritylists/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *OciSecurityListReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	sl := &ociv1beta1.OciSecurityList{}
	return r.Reconciler.Reconcile(ctx, req, sl)
}

// SetupWithManager sets up the controller with the Manager.
func (r *OciSecurityListReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ociv1beta1.OciSecurityList{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 3}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

// OciNetworkSecurityGroupReconciler reconciles an OciNetworkSecurityGroup object
type OciNetworkSecurityGroupReconciler struct {
	Reconciler *core.BaseReconciler
}

// +kubebuilder:rbac:groups=oci.oracle.com,resources=ocinetworksecuritygroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=oci.oracle.com,resources=ocinetworksecuritygroups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=oci.oracle.com,resources=ocinetworksecuritygroups/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *OciNetworkSecurityGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	nsg := &ociv1beta1.OciNetworkSecurityGroup{}
	return r.Reconciler.Reconcile(ctx, req, nsg)
}

// SetupWithManager sets up the controller with the Manager.
func (r *OciNetworkSecurityGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ociv1beta1.OciNetworkSecurityGroup{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 3}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

// OciRouteTableReconciler reconciles an OciRouteTable object
type OciRouteTableReconciler struct {
	Reconciler *core.BaseReconciler
}

// +kubebuilder:rbac:groups=oci.oracle.com,resources=ociroutetables,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=oci.oracle.com,resources=ociroutetables/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=oci.oracle.com,resources=ociroutetables/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *OciRouteTableReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	rt := &ociv1beta1.OciRouteTable{}
	return r.Reconciler.Reconcile(ctx, req, rt)
}

// SetupWithManager sets up the controller with the Manager.
func (r *OciRouteTableReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ociv1beta1.OciRouteTable{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 3}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
