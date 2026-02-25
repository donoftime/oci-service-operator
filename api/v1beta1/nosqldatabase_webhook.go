/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package v1beta1

import (
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var nosqldatabaselog = logf.Log.WithName("nosqldatabase-resource")

func (r *NoSQLDatabase) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-oci-oci-v1beta1-nosqldatabase,mutating=true,failurePolicy=ignore,sideEffects=None,groups=oci.oracle.com,resources=nosqldatabases,verbs=create;update,versions=v1,name=mnosqldatabase.kb.io,admissionReviewVersions={v1}

var _ webhook.Defaulter = &NoSQLDatabase{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *NoSQLDatabase) Default() {
	nosqldatabaselog.Info("default", "name", r.Name)
}

// +kubebuilder:webhook:path=/validate-oci-oci-v1beta1-nosqldatabase,mutating=false,failurePolicy=fail,sideEffects=None,groups=oci.oracle.com,resources=nosqldatabases,verbs=create;update,versions=v1,name=vnosqldatabase.kb.io,admissionReviewVersions={v1}

var _ webhook.Validator = &NoSQLDatabase{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *NoSQLDatabase) ValidateCreate() (admission.Warnings, error) {
	nosqldatabaselog.Info("validate create", "name", r.Name)
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *NoSQLDatabase) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	nosqldatabaselog.Info("validate update", "name", r.Name)
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *NoSQLDatabase) ValidateDelete() (admission.Warnings, error) {
	nosqldatabaselog.Info("validate delete", "name", r.Name)
	return nil, nil
}
