/*


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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/pkg/defaults"
	"github.com/jooho/nfs-provisioner-operator/pkg/reconciler"
)

// NFSProvisionerReconciler reconciles a NFSProvisioner object
type NFSProvisionerReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Reconciler reconciler.Reconciler
	Log        logr.Logger
}

// +kubebuilder:rbac:groups=cache.jhouse.com,resources=nfsprovisioners,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cache.jhouse.com,resources=nfsprovisioners/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cache.jhouse.com,resources=nfsprovisioners/finalizers,verbs=update
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=security.openshift.io,resources=securitycontextconstraints,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch
// +kubebuilder:rbac:groups=storage.k8s.io,resources=storageclasses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=endpoints,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=policy,resources=podsecuritypolicies,verbs=use
// +kubebuilder:rbac:groups=core,resources=persistentvolumes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is main method for operator
func (r *NFSProvisionerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("nfsprovisioner", req.NamespacedName)

	// Fetch the NFSProvisioner instance
	nfsprovisioner := &cachev1alpha1.NFSProvisioner{}
	err := r.Get(ctx, req.NamespacedName, nfsprovisioner)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("NFSProvisioner resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get NFSProvisioner")
		return ctrl.Result{}, err
	}

	// Handle deletion with finalizer
	const finalizerName = "nfsprovisioner.finalizers.jhouse.io"
	if nfsprovisioner.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted
		if !controllerutil.ContainsFinalizer(nfsprovisioner, finalizerName) {
			log.Info("Adding Finalizer for the NFSProvisioner")
			controllerutil.AddFinalizer(nfsprovisioner, finalizerName)
			if err := r.Update(ctx, nfsprovisioner); err != nil {
				log.Error(err, "Failed to update CR NFSProvisioner to add finalizer")
				return ctrl.Result{}, err
			}
			// Requeue immediately - the predicate filters metadata-only updates
			// so the Update won't trigger a new reconcile event automatically.
			return ctrl.Result{Requeue: true}, nil
		}
	} else {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(nfsprovisioner, finalizerName) {
			// Handle external resource deletion
			if err := r.deleteExternalResources(ctx, nfsprovisioner); err != nil {
				log.Error(err, "Failed to delete external resources")
				return ctrl.Result{}, err
			}

			// Remove finalizer
			log.Info("Removing Finalizer for the NFSProvisioner")
			controllerutil.RemoveFinalizer(nfsprovisioner, finalizerName)
			if err := r.Update(ctx, nfsprovisioner); err != nil {
				log.Error(err, "Failed to update CR NFSProvisioner to remove finalizer")
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	// Delegate to pkg/reconciler for main reconciliation logic
	return r.Reconciler.Reconcile(ctx, nfsprovisioner)
}

// Delete any external resources associated with the nfs server
func (r *NFSProvisionerReconciler) deleteExternalResources(ctx context.Context, m *cachev1alpha1.NFSProvisioner) error {
	log := r.Log.WithName("deleteExternalResource")
	// To-Do Delete all pvc that pawned by NFS SC

	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaults.ClusterRole,
		},
	}
	err := r.Get(ctx, types.NamespacedName{Name: defaults.ClusterRole, Namespace: ""}, clusterRole)
	if err == nil {
		log.Info("Deleting ClusterRole for NFSProvisioner")
		err = r.Delete(ctx, clusterRole, &client.DeleteOptions{})
		if err != nil {
			log.Error(err, "Failed to delete ClusterRole for NFSProvisioner", "ClusterRole.Name", defaults.ClusterRole)
			return err
		}
	}

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaults.ClusterRoleBinding,
		},
	}

	err = r.Get(ctx, types.NamespacedName{Name: defaults.ClusterRoleBinding, Namespace: ""}, clusterRoleBinding)
	if err == nil {
		log.Info("Deleting ClusterRoleBinding for NFSProvisioner")
		err = r.Delete(ctx, clusterRoleBinding, &client.DeleteOptions{})
		if err != nil {
			log.Error(err, "Failed to delete ClusterRoleBinding for NFSProvisioner", "ClusterRoleBinding.Name", defaults.ClusterRoleBinding)
			return err
		}
	}

	return nil
}


// SetupWithManager return error
func (r *NFSProvisionerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Filter for the primary resource (NFSProvisioner):
	// skip status-only updates, allow create/delete/spec changes.
	nfsPredicate := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1alpha1.NFSProvisioner{}, builder.WithPredicates(nfsPredicate)).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ServiceAccount{}).
		Complete(r)
}
