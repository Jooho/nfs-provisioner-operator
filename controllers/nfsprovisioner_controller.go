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
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/controllers/defaults"
	"github.com/jooho/nfs-provisioner-operator/controllers/resources"
)

// NFSProvisionerReconciler reconciles a NFSProvisioner object
type NFSProvisionerReconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	ResourceManager *resources.ResourceManagerSet
}

func validate(m *cachev1alpha1.NFSProvisioner) error {
	pvc := m.Spec.Pvc
	sc := m.Spec.SCForNFSPvc
	hostPathDir := m.Spec.HostPathDir
	if pvc != "" && (sc != "" || hostPathDir != "") {
		return fmt.Errorf("scForPvc or hostPathDir can not set with Pvc")
	}

	if hostPathDir != "" && (sc != "" || pvc != "") {
		return fmt.Errorf("scForPvc or Pvc can not set with hostPathDir")
	}

	if sc != "" && (pvc != "" || hostPathDir != "") {
		return fmt.Errorf("Pvc or hostPathDir can not set with scForPvc")
	}

	return nil
}

// +kubebuilder:rbac:groups=cache.jhouse.com,resources=nfsprovisioners,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cache.jhouse.com,resources=nfsprovisioners/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cache.jhouse.com,resources=nfsprovisioners/finalizers,verbs=update
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=security.openshift.io,resources=securitycontextconstraints,verbs=get;list;watch;create;update;patch;delete
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
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Validate checking
	if err = validate(nfsprovisioner); err != nil {
		log.Error(err, fmt.Sprintf("pvc: %s | sc: %s | hostPathDir: %s", nfsprovisioner.Spec.Pvc, nfsprovisioner.Spec.SCForNFSPvc, nfsprovisioner.Spec.HostPathDir))

		nfsprovisioner.Status.Error = fmt.Sprintf("pvc: %s | sc: %s | hostPathDir: %s", nfsprovisioner.Spec.Pvc, nfsprovisioner.Spec.SCForNFSPvc, nfsprovisioner.Spec.HostPathDir)
		nfsprovisioner.Status.Nodes = []string{}
		err := r.Status().Update(ctx, nfsprovisioner)
		if err != nil {
			log.Error(err, "Failed to update nfsprovisioner status")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, err
	}

	// Ensure required resources using resource managers
	if err := r.ResourceManager.EnsureAllResources(ctx, nfsprovisioner); err != nil {
		log.Error(err, "Failed to ensure required resources")
		return ctrl.Result{}, err
	}

	// Delete Logic
	// name of our custom finalizer
	const finalizerName = "nfsprovisioner.finalizers.jhouse.io"

	// examine DeletionTimestamp to determine if object is under deletion
	// isNFSProvisionerdMarkedToBeDeleted := nfsprovisioner.GetDeletionTimestamp() != nil

	// if nfsprovisioner.ObjectMeta.DeletionTimestamp.IsZero() {
	if nfsprovisioner.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.

		if !containsString(nfsprovisioner.GetFinalizers(), finalizerName) {
			log.Info("Adding Finalizer for the NFSProvisioner")

			controllerutil.AddFinalizer(nfsprovisioner, finalizerName)

			if err := r.Update(context.Background(), nfsprovisioner); err != nil {
				log.Error(err, "Failed to update CR NFSProvisioner to add finalizer")
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if containsString(nfsprovisioner.ObjectMeta.Finalizers, finalizerName) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.deleteExternalResources(nfsprovisioner); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried

				log.Error(err, "Failed to delete external resoureces")
				return ctrl.Result{}, err
			}

			// remove our finalizer from the list and update it.
			log.Info("Removing Finalizer for the NFSProvisioner")
			controllerutil.RemoveFinalizer(nfsprovisioner, finalizerName)

			if err := r.Update(context.Background(), nfsprovisioner); err != nil {
				log.Error(err, "Failed to update CR NFSProvisioner with finalizer to remove finalizer")
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	return ctrl.Result{Requeue: true}, nil
}

// Delete any external resources associated with the nfs server
func (r *NFSProvisionerReconciler) deleteExternalResources(m *cachev1alpha1.NFSProvisioner) error {
	log := r.Log.WithName("deleteExternalResource")
	// To-Do Delete all pvc that pawned by NFS SC
	ctx := context.Background()

	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaults.ClusterRole,
		}}
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
		}}

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

// Helper functions to check and remove string from a slice of strings.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

// SetupWithManager return error
func (r *NFSProvisionerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1alpha1.NFSProvisioner{}).
		Complete(r)

}
