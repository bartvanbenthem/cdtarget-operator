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
	"time"

	v1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

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
	logger := log.FromContext(ctx)

	// Fetch CDTarget object if it exists
	cdt := cnadv1alpha1.CDTarget{}
	err := r.Get(ctx, req.NamespacedName, &cdt)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("CDTarget resource not found. Ignoring since object must be deleted")
			// Exit reconciliation as the object has been deleted
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get CDTarget")
		// Requeue reconciliation as we were unable to fetch the object
		return ctrl.Result{}, err
	}

	// fetch ports from configmap
	var ports []int32
	cm := v1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{Name: "cdtarget-ports",
		Namespace: "cdtarget-operator-system"}, &cm)
	if err != nil && errors.IsNotFound(err) {
		logger.Error(err, "Failed to get ConfigMap cdtarget-ports")
		return ctrl.Result{}, err
	}
	ports, err = getPortsFromConfigMap(cm)
	if err != nil {
		logger.Error(err, "Failed to parse ports")
	}

	// Fetch NetworkPolicy object if it exists
	found := netv1.NetworkPolicy{}
	err = r.Get(ctx, types.NamespacedName{Name: cdt.Name, Namespace: cdt.Namespace}, &found)
	if err != nil && errors.IsNotFound(err) {
		net := r.networkPolicyForCDTarget(&cdt, ports)
		logger.Info("Creating a new NetworkPolicy", "NetworkPolicy.Namespace",
			net.Namespace, "NetworkPolicy.Name", net.Name)
		err = r.Create(ctx, net)
		if err != nil {
			logger.Error(err, "Failed to create new NetworkPolicy",
				"NetworkPolicy.Namespace", net.Namespace,
				"NetworkPolicy.Name", net.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	} else if err != nil {
		logger.Error(err, "Failed to get NetworkPolicy")
		return ctrl.Result{}, err
	} else {
		net := r.networkPolicyForCDTarget(&cdt, ports)
		logger.Info("Updating NetworkPolicy", "NetworkPolicy.Namespace",
			net.Namespace, "NetworkPolicy.Name", net.Name)
		r.Update(ctx, net)
		if err != nil {
			logger.Error(err, "Failed to Update NetworkPolicy",
				"NetworkPolicy.Namespace", net.Namespace,
				"NetworkPolicy.Name", net.Name)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CDTargetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cnadv1alpha1.CDTarget{}).
		Owns(&netv1.NetworkPolicy{}).
		Complete(r)
}
