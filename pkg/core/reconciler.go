/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package core

import (
	"context"
	"fmt"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/metrics"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/util"
)

const (
	OSOKFinalizerName  = "finalizers.oci.oracle.com/oci-resources"
	defaultRequeueTime = time.Minute * 2
)

type BaseReconciler struct {
	client.Client
	OSOKServiceManager   servicemanager.OSOKServiceManager
	Finalizer            Finalizer
	Log                  loggerutil.OSOKLogger
	Metrics              *metrics.Metrics
	Recorder             record.EventRecorder
	Scheme               *runtime.Scheme
	AdditionalFinalizers []string
}

func (r *BaseReconciler) Reconcile(ctx context.Context, req ctrl.Request, obj client.Object) (result ctrl.Result, err error) {
	// To setup the fixed logs for every log
	ctx = metrics.AddFixedLogMapEntries(ctx, req.Name, req.Namespace)
	if result, stop, err := r.fetchResource(ctx, req, obj); stop {
		return result, err
	}
	if result, stop, err := r.handleDeletion(ctx, req, obj); stop {
		return result, err
	}
	if result, stop, err := r.ensureFinalizers(ctx, req, obj); stop {
		return result, err
	}

	r.Log.InfoLogWithFixedMessage(ctx, "Reconcile the resource")
	return r.ReconcileResource(ctx, obj, req)
}

func (r *BaseReconciler) fetchResource(ctx context.Context, req ctrl.Request, obj client.Object) (ctrl.Result, bool, error) {
	r.Log.DebugLogWithFixedMessage(ctx, "Fetching the resource from server")
	if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
		if errors.IsNotFound(err) {
			r.Log.ErrorLogWithFixedMessage(ctx, err, "The resource could be in deleting state. Ignoring")
			return ctrl.Result{}, true, client.IgnoreNotFound(err)
		}
		r.Log.ErrorLogWithFixedMessage(ctx, err, "Error while get the Resource from server.")
		return ctrl.Result{}, true, err
	}

	r.Log.InfoLogWithFixedMessage(ctx, "Got the status of resource")
	return ctrl.Result{}, false, nil
}

func (r *BaseReconciler) handleDeletion(ctx context.Context, req ctrl.Request, obj client.Object) (ctrl.Result, bool, error) {
	if obj.GetDeletionTimestamp() == nil || !controllerutil.ContainsFinalizer(obj, OSOKFinalizerName) {
		return ctrl.Result{}, false, nil
	}

	r.Log.InfoLogWithFixedMessage(ctx, "The Deletion time is non zero. Deleting the resource")
	deleteSucceeded, err := r.DeleteResource(ctx, obj, req)
	if err != nil {
		return r.deleteFailureResult(ctx, req, obj, err)
	}
	if !deleteSucceeded {
		return r.deleteRetryResult(ctx, req, obj)
	}

	return r.deleteSuccessResult(ctx, req, obj)
}

func (r *BaseReconciler) ensureFinalizers(ctx context.Context, req ctrl.Request, obj client.Object) (ctrl.Result, bool, error) {
	if err := r.addFinalizer(ctx, obj, strings.Join(r.AdditionalFinalizers, " "), OSOKFinalizerName); err != nil {
		r.Log.ErrorLogWithFixedMessage(ctx, err, "Error adding finalizer to Custom Resource.")
		r.Metrics.AddReconcileFaultMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
			"Error adding finalizer to Custom Resource.", req.Name, req.Namespace)
		r.Recorder.Event(obj, v1.EventTypeWarning, "Failed", "Failed to add finalizer")
		result, requeueErr := util.RequeueWithError(ctx, err, defaultRequeueTime, r.Log)
		return result, true, requeueErr
	}

	return ctrl.Result{}, false, nil
}

func (r *BaseReconciler) deleteFailureResult(ctx context.Context, req ctrl.Request, obj client.Object, err error) (ctrl.Result, bool, error) {
	r.Log.ErrorLogWithFixedMessage(ctx, err, "Requeuing object due to error during delete of CR")
	r.Metrics.AddCRDeleteFaultMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
		"Requeuing object due to error during delete of CR", req.Name, req.Namespace)
	r.Recorder.Event(obj, v1.EventTypeWarning, "Failed",
		fmt.Sprintf("Failed to remove the finalizer: %s", err.Error()))
	result, requeueErr := util.RequeueWithError(ctx, err, defaultRequeueTime, r.Log)
	return result, true, requeueErr
}

func (r *BaseReconciler) deleteRetryResult(ctx context.Context, req ctrl.Request, obj client.Object) (ctrl.Result, bool, error) {
	r.Log.InfoLogWithFixedMessage(ctx, "Re-queuing object as delete was unsuccessful")
	r.Metrics.AddCRDeleteFaultMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
		"Re-queuing object as delete was unsuccessful", req.Name, req.Namespace)
	r.Recorder.Event(obj, v1.EventTypeWarning, "Failed", "Failed Delete the resource")
	result, err := util.RequeueWithoutError(ctx, defaultRequeueTime, r.Log)
	return result, true, err
}

func (r *BaseReconciler) deleteSuccessResult(ctx context.Context, req ctrl.Request, obj client.Object) (ctrl.Result, bool, error) {
	if err := r.removeFinalizer(ctx, obj, strings.Join(r.AdditionalFinalizers, " "), OSOKFinalizerName); err != nil {
		r.Log.ErrorLogWithFixedMessage(ctx, err, "Failed to remove the finalizer")
		r.Recorder.Event(obj, v1.EventTypeWarning, "Failed",
			fmt.Sprintf("Failed to remove the finalizer: %s", err.Error()))
		result, requeueErr := util.RequeueWithError(ctx, err, defaultRequeueTime, r.Log)
		return result, true, requeueErr
	}

	r.Log.InfoLogWithFixedMessage(ctx, "Deletion of the CR successful")
	r.Metrics.AddCRDeleteSuccessMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
		"Deletion of the CR successful", req.Name, req.Namespace)
	r.Recorder.Event(obj, v1.EventTypeNormal, "Success", "Removed finalizer")
	result, err := util.DoNotRequeue()
	return result, true, err
}

func (r *BaseReconciler) GetStatus(obj client.Object) (*v1beta1.OSOKStatus, error) {
	status, err := r.OSOKServiceManager.GetCrdStatus(obj)
	if err != nil {
		return nil, err
	}

	if status.RequestedAt == nil {
		now := metav1.NewTime(time.Now())
		status.RequestedAt = &now
	}

	return status, nil
}

func (r *BaseReconciler) ReconcileResource(ctx context.Context, obj client.Object, req ctrl.Request) (ctrl.Result, error) {
	ctx = metrics.AddFixedLogMapEntries(ctx, req.Name, req.Namespace)

	oldObj := obj.DeepCopyObject().(client.Object)
	OSOKResponse, err := r.OSOKServiceManager.CreateOrUpdate(ctx, obj, req)
	if err != nil {
		r.Log.ErrorLogWithFixedMessage(ctx, err, "Create Or Update failed in the Service Manager with error")
		r.Metrics.AddReconcileFaultMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
			"Create Or Update failed in the Service Manager", req.Name, req.Namespace)
		r.Recorder.Event(obj, v1.EventTypeWarning, "Failed",
			fmt.Sprintf("Failed to create or update resource: %s", err.Error()))
	}

	if err := r.Status().Patch(ctx, obj, client.MergeFrom(oldObj)); err != nil {
		r.Log.ErrorLogWithFixedMessage(ctx, err, "Error updating the status of the Object")
		r.Metrics.AddReconcileFaultMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
			"Error updating the status of the CR", req.Name, req.Namespace)
		r.Recorder.Event(obj, v1.EventTypeWarning, "Failed",
			fmt.Sprintf("Failed to create or update resource: %s", err.Error()))
		return util.RequeueWithError(ctx, err, defaultRequeueTime, r.Log)
	}
	r.Metrics.AddCRCountMetrics(ctx, r.Metrics.ServiceName, "Created an Custom resource "+r.Metrics.ServiceName,
		req.Name, req.Namespace)

	if OSOKResponse.IsSuccessful {
		r.Log.InfoLogWithFixedMessage(ctx, "Reconcile Completed")
		r.Metrics.AddReconcileSuccessMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
			"Create or Update of resource succeeded", req.Name, req.Namespace)
		r.Recorder.Event(obj, v1.EventTypeNormal, "Success", "Create or Update of resource succeeded")
		if OSOKResponse.ShouldRequeue {
			return r.requeueResult(ctx, OSOKResponse, nil)
		}
		return util.DoNotRequeue()
	} else {
		r.Log.InfoLogWithFixedMessage(ctx, "Reconcile Failed")
		r.Metrics.AddReconcileFaultMetrics(ctx, obj.GetObjectKind().GroupVersionKind().Kind,
			"Failed to create or update resource", req.Name, req.Namespace)
		r.Recorder.Event(obj, v1.EventTypeWarning, "Failed", "Failed to create or update resource")
		if OSOKResponse.ShouldRequeue {
			return r.requeueResult(ctx, OSOKResponse, err)
		}
		return util.DoNotRequeue()
	}
}

func (r *BaseReconciler) requeueResult(ctx context.Context, response servicemanager.OSOKResponse, err error) (ctrl.Result, error) {
	duration := response.RequeueDuration
	if duration <= 0 {
		duration = defaultRequeueTime
	}

	if err != nil {
		return util.RequeueWithError(ctx, err, duration, r.Log)
	}

	if duration > 0 {
		return util.RequeueWithoutError(ctx, duration, r.Log)
	}

	return ctrl.Result{Requeue: true}, nil
}

func (r *BaseReconciler) DeleteResource(ctx context.Context, obj client.Object, req ctrl.Request) (bool, error) {
	ctx = metrics.AddFixedLogMapEntries(ctx, req.Name, req.Namespace)
	//log := util.LogUtil{Log: r.Log.WithValues("name", req.Name, "namespace", req.Namespace)}
	//TODO Emit Delete Start metrics
	delSucc, err := r.OSOKServiceManager.Delete(ctx, obj)
	if err != nil {
		r.Log.ErrorLogWithFixedMessage(ctx, err, "Delete failed in the Service Manager with error", "name", req.Name,
			"namespace", req.Namespace, "namespacedName", req.String())
		r.Recorder.Event(obj, v1.EventTypeWarning, "Failed",
			fmt.Sprintf("Failed to delete resource: %s", err.Error()))
		// TODO Emit Delete Fault metrics end
		return false, err
	}
	if delSucc {
		r.Log.InfoLogWithFixedMessage(ctx, "Delete Successful")
	} else {
		r.Log.InfoLogWithFixedMessage(ctx, "Delete Unsuccessful, re-queuing the request after 2 minutes")
		r.Recorder.Event(obj, v1.EventTypeWarning, "Failed", "Delete Unsuccessful")
	}
	// TODO Emit Delete Success metrics end
	return delSucc, nil
}

func (r *BaseReconciler) addFinalizer(ctx context.Context, obj client.Object, finalizers ...string) error {
	needsUpdate := false
	for _, finalizer := range finalizers {
		if finalizer != "" && !controllerutil.ContainsFinalizer(obj, finalizer) {
			controllerutil.AddFinalizer(obj, finalizer)
			needsUpdate = true
		}
	}
	if !needsUpdate {
		return nil
	}
	r.Log.InfoLogWithFixedMessage(ctx, "Added Finalizer to the resource.")
	r.Recorder.Event(obj, v1.EventTypeNormal, "Success", "Finalizer is added to the object")
	return r.Update(ctx, obj)
}

func (r *BaseReconciler) removeFinalizer(ctx context.Context, obj client.Object, finalizers ...string) error {
	needsUpdate := false
	for _, finalizer := range finalizers {
		if finalizer != "" && controllerutil.ContainsFinalizer(obj, finalizer) {
			controllerutil.RemoveFinalizer(obj, finalizer)
			needsUpdate = true
		}
	}
	if !needsUpdate {
		return nil
	}
	r.Log.InfoLogWithFixedMessage(ctx, "Removing Finalizer from the resource.")
	r.Recorder.Event(obj, v1.EventTypeNormal, "Success", "Finalizer is removed from the object")
	return r.Update(ctx, obj)
}
