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
	kedav2 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	apiv2 "github.com/operator-framework/api/pkg/operators/v2"
	"github.com/operator-framework/operator-lib/conditions"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operators.coreos.com,resources=operatorconditions,verbs=get;list;watch
//+kubebuilder:rbac:groups=keda.sh,resources=scaledobjects,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=keda.sh,resources=triggerauthentications,verbs=get;list;watch;create;update;patch;delete

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
		logger.Info("Operator CDTarget resource object not found.")
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

	// Fetch CDTarget token secret object if it exists
	// Only if it does not exist create the token secret
	// so token values can be added later to enable token functionality
	// the controller is not an owner after initial creation
	token := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Name: operatorCR.Spec.TokenRef,
		Namespace: operatorCR.Namespace}, token)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Existing Token Secret Not Found")
		logger.Info("Creating Token Secret, add values later to enable azure pipeline Auth")
		token = r.tokenSecretForCDTarget(operatorCR)
		err = r.Create(ctx, token)
		if err != nil {
			meta.SetStatusCondition(&operatorCR.Status.Conditions, metav1.Condition{
				Type:               "ReconcileSuccess",
				Status:             metav1.ConditionFalse,
				Reason:             cnadv1alpha1.ReasonOperandSecretFailed,
				LastTransitionTime: metav1.NewTime(time.Now()),
				Message:            fmt.Sprintf("unable to update operand Token Secret: %s", err.Error()),
			})
			return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, operatorCR)})
		}
	} else if err != nil {
		logger.Error(err, "Error getting operator CDTarget Token Secret object")
		meta.SetStatusCondition(&operatorCR.Status.Conditions, metav1.Condition{
			Type:               "ReconcileSuccess",
			Status:             metav1.ConditionFalse,
			Reason:             cnadv1alpha1.ReasonSecretNotAvailable,
			LastTransitionTime: metav1.NewTime(time.Now()),
			Message:            fmt.Sprintf("unable to configure Token Secret: %s", err.Error()),
		})
		return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, operatorCR)})
	}

	// Fetch TriggerAuthentication object if it exists
	// Only if it does not exist create the TriggerAuthentication
	// the controller is not an owner after initial creation
	trigauth := &kedav2.TriggerAuthentication{}
	err = r.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-trigger-auth", operatorCR.Spec.Config.PoolName),
		Namespace: operatorCR.Namespace}, trigauth)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Existing TriggerAuthentication Not Found")
		logger.Info("Creating TriggerAuthentication")
		trigauth = r.triggerAuthenticationForCDTarget(operatorCR)
		err = r.Create(ctx, trigauth)
		if err != nil {
			meta.SetStatusCondition(&operatorCR.Status.Conditions, metav1.Condition{
				Type:               "ReconcileSuccess",
				Status:             metav1.ConditionFalse,
				Reason:             cnadv1alpha1.ReasonOperandTriggerAuthenticationFailed,
				LastTransitionTime: metav1.NewTime(time.Now()),
				Message:            fmt.Sprintf("unable to update operand TriggerAuthentication: %s", err.Error()),
			})
			return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, operatorCR)})
		}
	} else if err != nil {
		logger.Error(err, "Error getting existing CDTArget TriggerAuthentication.")
		meta.SetStatusCondition(&operatorCR.Status.Conditions, metav1.Condition{
			Type:               "ReconcileSuccess",
			Status:             metav1.ConditionFalse,
			Reason:             cnadv1alpha1.ReasonTriggerAuthenticationNotAvailable,
			LastTransitionTime: metav1.NewTime(time.Now()),
			Message:            fmt.Sprintf("unable to get operand TriggerAuthentication: %s", err.Error()),
		})
		return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, operatorCR)})
	}

	// Fetch ConfigMap cdtarget-ports object from operator namespace if it exists
	cmport := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{Name: "cdtarget-ports",
		Namespace: "cdtarget-operator"}, cmport)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Existing ConfigMap cdtarget-ports Not Found")
		logger.Info("Creating ConfigMap cdtarget-ports from assets manifests")
		cmport = assets.GetConfigMapFromFile("manifests/cdtarget_ports.yaml")
		err = r.Create(ctx, cmport)
		if err != nil {
			meta.SetStatusCondition(&operatorCR.Status.Conditions, metav1.Condition{
				Type:               "ReconcileSuccess",
				Status:             metav1.ConditionFalse,
				Reason:             cnadv1alpha1.ReasonOperandConfigMapFailed,
				LastTransitionTime: metav1.NewTime(time.Now()),
				Message:            fmt.Sprintf("unable to update operand cdtarget-ports Configmap: %s", err.Error()),
			})
			return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, operatorCR)})
		}
	} else if err != nil {
		logger.Error(err, "Error getting operator CDTarget ConfigMap object")
		meta.SetStatusCondition(&operatorCR.Status.Conditions, metav1.Condition{
			Type:               "ReconcileSuccess",
			Status:             metav1.ConditionFalse,
			Reason:             cnadv1alpha1.ReasonConfigMapNotAvailable,
			LastTransitionTime: metav1.NewTime(time.Now()),
			Message:            fmt.Sprintf("unable to configure ConfigMap cdtarget-ports: %s", err.Error()),
		})
		return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, operatorCR)})
	}

	// Fetch NetworkPolicy CDTarget Egress object if it exists
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

	// Fetch ports from ConfigMap
	ports, err := getPortsFromConfigMap(cmport)
	if err != nil {
		logger.Error(err, "Failed to parse ports")
	}

	netpol = r.networkPolicyForCDTarget(operatorCR, ports)
	if err = ctrl.SetControllerReference(operatorCR, netpol, r.Scheme); err != nil {
		logger.Error(err, "Failed to set NetworkPolicy controller reference")
		return ctrl.Result{}, err
	}

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

	// Fetch NetworkPolicy object azure-pipelines-pool if it exists
	azp := &netv1.NetworkPolicy{}
	create = false
	spoolname := fmt.Sprintf("%s-pool", operatorCR.Name)
	err = r.Get(ctx, types.NamespacedName{Name: spoolname,
		Namespace: operatorCR.Namespace}, azp)
	if err != nil && errors.IsNotFound(err) {
		logger.Info(fmt.Sprintf("Existing NetworkPolicy %s Not Found", spoolname))
		logger.Info(fmt.Sprintf("Creating NetworkPolicy %s from assets manifests", spoolname))
		create = true
	} else if err != nil {
		logger.Error(err, fmt.Sprintf("Error getting existing CDTArget NetworkPolicy %s.", spoolname))
		meta.SetStatusCondition(&operatorCR.Status.Conditions, metav1.Condition{
			Type:               "ReconcileSuccess",
			Status:             metav1.ConditionFalse,
			Reason:             cnadv1alpha1.ReasonNetworkPolicyNotAvailable,
			LastTransitionTime: metav1.NewTime(time.Now()),
			Message:            fmt.Sprintf("unable to get operand NetworkPolicy %s: %s", spoolname, err.Error()),
		})
		return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, operatorCR)})
	}

	azp = assets.GetNetworkPolicyFromFile("manifests/az-pipelines-pool.yaml")
	azp.ObjectMeta.Name = spoolname
	azp.ObjectMeta.Namespace = operatorCR.Namespace
	azp.ObjectMeta.Labels = operatorCR.Spec.AdditionalSelector
	azp.Spec.PodSelector.MatchLabels = operatorCR.Spec.AdditionalSelector
	if err = ctrl.SetControllerReference(operatorCR, azp, r.Scheme); err != nil {
		logger.Error(err, "Failed to set NetworkPolicy controller reference")
		return ctrl.Result{}, err
	}

	if create {
		err = r.Create(ctx, azp)
	} else {
		err = r.Update(ctx, azp)
	}

	if err != nil {
		meta.SetStatusCondition(&operatorCR.Status.Conditions, metav1.Condition{
			Type:               "ReconcileSuccess",
			Status:             metav1.ConditionFalse,
			Reason:             cnadv1alpha1.ReasonOperandNetworkPolicyFailed,
			LastTransitionTime: metav1.NewTime(time.Now()),
			Message:            fmt.Sprintf("unable to update operand NetworkPolicy azure-pipelines-pool: %s", err.Error()),
		})
		return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, operatorCR)})
	}

	// Fetch cdtarget-config ConfigMap object if it exists
	cmcfg := &corev1.ConfigMap{}
	create = false
	cfgname := fmt.Sprintf("%s-config", operatorCR.Name)
	err = r.Get(ctx, types.NamespacedName{Name: cfgname, Namespace: operatorCR.Namespace}, cmcfg)
	if err != nil && errors.IsNotFound(err) {
		create = true
	} else if err != nil {
		logger.Error(err, "Error getting CDTArget Config")
		meta.SetStatusCondition(&operatorCR.Status.Conditions, metav1.Condition{
			Type:               "ReconcileSuccess",
			Status:             metav1.ConditionFalse,
			Reason:             cnadv1alpha1.ReasonConfigMapNotAvailable,
			LastTransitionTime: metav1.NewTime(time.Now()),
			Message:            fmt.Sprintf("unable to get operand Config: %s", err.Error()),
		})
		return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, operatorCR)})
	}

	cmcfg = r.configMapForCDTarget(operatorCR)
	if err = ctrl.SetControllerReference(operatorCR, cmcfg, r.Scheme); err != nil {
		logger.Error(err, "Failed to set ConfigMap controller reference")
		return ctrl.Result{}, err
	}

	if create {
		err = r.Create(ctx, cmcfg)
	} else {
		err = r.Update(ctx, cmcfg)
	}

	if err != nil {
		meta.SetStatusCondition(&operatorCR.Status.Conditions, metav1.Condition{
			Type:               "ReconcileSuccess",
			Status:             metav1.ConditionFalse,
			Reason:             cnadv1alpha1.ReasonOperandConfigMapFailed,
			LastTransitionTime: metav1.NewTime(time.Now()),
			Message:            fmt.Sprintf("unable to update operand ConfigMap: %s", err.Error()),
		})
		return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, operatorCR)})
	}

	// Fetch agent Deployment object if it exists
	// After creation the Deployment is never updated by the operator
	// to avoid conflicts with the horizontal pod scaler & KEDA
	// The operator does own the deployment object for re-creation
	deployment := &appsv1.Deployment{}
	create = false
	err = r.Get(ctx, types.NamespacedName{Name: operatorCR.Name, Namespace: operatorCR.Namespace}, deployment)
	if err != nil && errors.IsNotFound(err) {
		create = true
	} else if err != nil {
		logger.Error(err, "Error getting CDTArget Agent Deployment.")
		meta.SetStatusCondition(&operatorCR.Status.Conditions, metav1.Condition{
			Type:               "ReconcileSuccess",
			Status:             metav1.ConditionFalse,
			Reason:             cnadv1alpha1.ReasonDeploymentNotAvailable,
			LastTransitionTime: metav1.NewTime(time.Now()),
			Message:            fmt.Sprintf("unable to get operand Deployment: %s", err.Error()),
		})
		return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, operatorCR)})
	}

	deployment = r.deploymentForCDTarget(operatorCR)
	if err = ctrl.SetControllerReference(operatorCR, deployment, r.Scheme); err != nil {
		logger.Error(err, "Failed to set Deployment controller reference")
		return ctrl.Result{}, err
	}

	if create {
		logger.Info(fmt.Sprintf("Creating Deployment %s", deployment.Name))
		err = r.Create(ctx, deployment)
	}

	if err != nil {
		meta.SetStatusCondition(&operatorCR.Status.Conditions, metav1.Condition{
			Type:               "ReconcileSuccess",
			Status:             metav1.ConditionFalse,
			Reason:             cnadv1alpha1.ReasonOperandDeploymentFailed,
			LastTransitionTime: metav1.NewTime(time.Now()),
			Message:            fmt.Sprintf("unable to create operand Deployment: %s", err.Error()),
		})
		return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, operatorCR)})
	}

	// Fetch ScaledObject if it exists
	// After creation the ScaledObject is never updated by the operator
	// The operator does own the ScaledObject object for re-creation
	so := &kedav2.ScaledObject{}
	create = false
	err = r.Get(ctx, types.NamespacedName{Name: operatorCR.Name, Namespace: operatorCR.Namespace}, so)
	if err != nil && errors.IsNotFound(err) {
		create = true
	} else if err != nil {
		logger.Error(err, "Error getting CDTArget Agent ScaledObject.")
		meta.SetStatusCondition(&operatorCR.Status.Conditions, metav1.Condition{
			Type:               "ReconcileSuccess",
			Status:             metav1.ConditionFalse,
			Reason:             cnadv1alpha1.ReasonScaledObjectNotAvailable,
			LastTransitionTime: metav1.NewTime(time.Now()),
			Message:            fmt.Sprintf("unable to get operand ScaledObject: %s", err.Error()),
		})
		return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, operatorCR)})
	}

	so = r.scaledObjectForCDTarget(operatorCR)
	if err = ctrl.SetControllerReference(operatorCR, so, r.Scheme); err != nil {
		logger.Error(err, "Failed to set ScaledObject controller reference")
		return ctrl.Result{}, err
	}

	if create {
		logger.Info(fmt.Sprintf("Creating ScaledObject %s", so.Name))
		err = r.Create(ctx, so)
	}

	if err != nil {
		meta.SetStatusCondition(&operatorCR.Status.Conditions, metav1.Condition{
			Type:               "ReconcileSuccess",
			Status:             metav1.ConditionFalse,
			Reason:             cnadv1alpha1.ReasonOperandScaledObjectFailed,
			LastTransitionTime: metav1.NewTime(time.Now()),
			Message:            fmt.Sprintf("unable to create operand ScaledObject: %s", err.Error()),
		})
		return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, operatorCR)})
	}

	// Finalize reconcile loop and set succesfull status condition
	meta.SetStatusCondition(&operatorCR.Status.Conditions, metav1.Condition{
		Type:               "ReconcileSuccess",
		Status:             metav1.ConditionTrue,
		Reason:             cnadv1alpha1.ReasonSucceeded,
		LastTransitionTime: metav1.NewTime(time.Now()),
		Message:            "operator successfully reconciling",
	})
	r.Status().Update(ctx, operatorCR)

	// OLM condition reporting
	cdo := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: "cdtarget-controller-manager",
		Namespace: "cdtarget-operator"}, cdo)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("cdtarget-controller-manager not found")
	} else if err != nil {
		logger.Error(err, "Error fetching CDTarget operator deployment")
	}

	if len(cdo.ObjectMeta.Annotations["olm.owner"]) != 0 {
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
	}

	return ctrl.Result{}, utilerrors.NewAggregate([]error{err, r.Status().Update(ctx, operatorCR)})

}

// SetupWithManager sets up the controller with the Manager.
func (r *CDTargetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cnadv1alpha1.CDTarget{}).
		Owns(&netv1.NetworkPolicy{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&kedav2.ScaledObject{}).
		Owns(&kedav2.TriggerAuthentication{}).
		Complete(r)
}
