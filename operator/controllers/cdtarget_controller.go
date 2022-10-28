/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/bartvanbenthem/cdtarget-operator/assets"
	"github.com/bartvanbenthem/cdtarget-operator/controllers/metrics"
	apiv2 "github.com/operator-framework/api/pkg/operators/v2"
	"github.com/operator-framework/operator-lib/conditions"
	v1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cnadv1alpha1 "github.com/bartvanbenthem/cdtarget-operator/api/v1alpha1"
)

// CDTargetReconciler reconciles a CDTarget object
type CDTargetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=cnad.gofound.nl,resources=cdtargets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cnad.gofound.nl,resources=cdtargets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cnad.gofound.nl,resources=cdtargets/finalizers,verbs=update
//+kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CDTarget object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *CDTargetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	metrics.ReconcilesTotal.Inc()
	logger := log.FromContext(ctx)

	// Fetch CDTarget object if it exists
	operatorCR := &cnadv1alpha1.CDTarget{}
	err := r.Get(ctx, req.NamespacedName, operatorCR)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Operator CDTArget resource object not found.")
		return ctrl.Result{}, nil
	} else if err != nil {
		logger.Error(err, "Error getting operator CDTarget resource object")
		meta.SetStatusCondition(&operatorCR.Status.Conditions, metav1.Condition{
			Type:               "ReconcileSuccess",
			Status:             metav1.ConditionFalse,
			Reason:             cnadv1alpha1.ReasonCRNotAvailable,
			LastTransitionTime: metav1.NewTime(time.Now()),
			Message:            fmt.Sprintf("unable to get operator custom resource: %s", err.Error()),
		})
		return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, operatorCR)})
	}

	// Fetch ConfigMap object if it exists
	cm := &v1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{Name: "cdtarget-ports",
		Namespace: "cdtarget-operator"}, cm)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Failed getting existing ConfigMap cdtarget-ports")
		logger.Info("Creating ConfigMap cdtarget-ports from assets manifests")
		cm = assets.GetConfigMapFromFile("manifests/cdtarget_ports.yaml")
		err = r.Create(ctx, cm)
		if err != nil {
			return ctrl.Result{}, err
		}
	} else if err != nil {
		logger.Error(err, "Error getting operator CDTarget resource object")
		meta.SetStatusCondition(&operatorCR.Status.Conditions, metav1.Condition{
			Type:               "ReconcileSuccess",
			Status:             metav1.ConditionFalse,
			Reason:             cnadv1alpha1.ReasonConfigMapNotAvailable,
			LastTransitionTime: metav1.NewTime(time.Now()),
			Message:            fmt.Sprintf("unable to configure ConfigMap cdtarget-ports: %s", err.Error()),
		})
		return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, operatorCR)})
	}
	// fetch ports from configmap
	ports, err := getPortsFromConfigMap(cm)
	if err != nil {
		logger.Error(err, "Failed to parse ports")
	}

	// Fetch NetworkPolicy object if it exists
	netpol := &netv1.NetworkPolicy{}
	create := false
	err = r.Get(ctx, types.NamespacedName{Name: operatorCR.Name, Namespace: operatorCR.Namespace}, netpol)
	if err != nil && errors.IsNotFound(err) {
		create = true
	} else if err != nil {
		logger.Error(err, "Error getting existing CDTArget NetworkPolicy.")
		meta.SetStatusCondition(&operatorCR.Status.Conditions, metav1.Condition{
			Type:               "ReconcileSuccess",
			Status:             metav1.ConditionFalse,
			Reason:             cnadv1alpha1.ReasonNetworkPolicyNotAvailable,
			LastTransitionTime: metav1.NewTime(time.Now()),
			Message:            fmt.Sprintf("unable to get operand NetworkPolicy: %s", err.Error()),
		})
		return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, operatorCR)})
	}

	netpol = r.networkPolicyForCDTarget(operatorCR, ports)
	ctrl.SetControllerReference(operatorCR, netpol, r.Scheme)

	if create {
		err = r.Create(ctx, netpol)
	} else {
		err = r.Update(ctx, netpol)
	}

	if err != nil {
		meta.SetStatusCondition(&operatorCR.Status.Conditions, metav1.Condition{
			Type:               "ReconcileSuccess",
			Status:             metav1.ConditionFalse,
			Reason:             cnadv1alpha1.ReasonOperandNetworkPolicyFailed,
			LastTransitionTime: metav1.NewTime(time.Now()),
			Message:            fmt.Sprintf("unable to update operand NetworkPolicy: %s", err.Error()),
		})
		return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, operatorCR)})
	}

	meta.SetStatusCondition(&operatorCR.Status.Conditions, metav1.Condition{
		Type:               "ReconcileSuccess",
		Status:             metav1.ConditionTrue,
		Reason:             cnadv1alpha1.ReasonSucceeded,
		LastTransitionTime: metav1.NewTime(time.Now()),
		Message:            "operator successfully reconciling",
	})
	r.Status().Update(ctx, operatorCR)

	// OLM condition reporting
	condition, err := conditions.InClusterFactory{Client: r.Client}.
		NewCondition(apiv2.ConditionType(apiv2.Upgradeable))

	if err != nil {
		return ctrl.Result{}, err
	}

	err = condition.Set(ctx, metav1.ConditionTrue,
		conditions.WithReason("OperatorUpgradeable"),
		conditions.WithMessage("The operator is currently upgradeable"))
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, operatorCR)})

}

// SetupWithManager sets up the controller with the Manager.
func (r *CDTargetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cnadv1alpha1.CDTarget{}).
		Owns(&netv1.NetworkPolicy{}).
		Complete(r)
}
