package resources

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/pkg/builder"
	"github.com/jooho/nfs-provisioner-operator/pkg/defaults"
)

// RBACManager manages all RBAC resources (ClusterRole, ClusterRoleBinding, Role, RoleBinding)
type RBACManager struct {
	BaseResourceManager
}

// NewRBACManager creates a new RBACManager
func NewRBACManager(base BaseResourceManager) *RBACManager {
	return &RBACManager{
		BaseResourceManager: base,
	}
}

// GetResourceName returns the name of the resource this manager handles
func (m *RBACManager) GetResourceName() string {
	return "RBAC"
}

// EnsureResource ensures all RBAC resources exist
func (m *RBACManager) EnsureResource(ctx context.Context, nfsProvisioner *cachev1alpha1.NFSProvisioner) error {
	log := m.Log.WithValues(
		"resource", m.GetResourceName(),
		"nfsprovisioner.name", nfsProvisioner.Name,
		"nfsprovisioner.namespace", nfsProvisioner.Namespace,
	)

	// Ensure ClusterRole
	if err := m.ensureClusterRole(ctx, nfsProvisioner); err != nil {
		log.Error(err, "Failed to ensure ClusterRole")
		return err
	}

	// Ensure ClusterRoleBinding
	if err := m.ensureClusterRoleBinding(ctx, nfsProvisioner); err != nil {
		log.Error(err, "Failed to ensure ClusterRoleBinding")
		return err
	}

	// Ensure Role
	if err := m.ensureRole(ctx, nfsProvisioner); err != nil {
		log.Error(err, "Failed to ensure Role")
		return err
	}

	// Ensure RoleBinding
	if err := m.ensureRoleBinding(ctx, nfsProvisioner); err != nil {
		log.Error(err, "Failed to ensure RoleBinding")
		return err
	}

	return nil
}

// ensureClusterRole ensures the ClusterRole exists
func (m *RBACManager) ensureClusterRole(ctx context.Context, nfsProvisioner *cachev1alpha1.NFSProvisioner) error {
	crFound := &rbacv1.ClusterRole{}
	err := m.Client.Get(ctx, types.NamespacedName{Name: defaults.ClusterRole, Namespace: ""}, crFound)
	if err != nil && errors.IsNotFound(err) {
		// Build ClusterRole using builder
		cr := builder.BuildClusterRole()
		m.Log.Info("Creating ClusterRole", "clusterrole.name", cr.Name)

		if err := m.Client.Create(ctx, cr); err != nil {
			m.Log.Error(err, "Failed to create ClusterRole", "clusterrole.name", cr.Name)
			return err
		}
		m.Log.Info("Successfully created ClusterRole", "clusterrole.name", cr.Name)
	} else if err != nil {
		m.Log.Error(err, "Failed to get ClusterRole")
		return err
	} else {
		m.Log.V(1).Info("ClusterRole already exists", "clusterrole.name", crFound.Name)
	}
	return nil
}

// ensureClusterRoleBinding ensures the ClusterRoleBinding exists
func (m *RBACManager) ensureClusterRoleBinding(ctx context.Context, nfsProvisioner *cachev1alpha1.NFSProvisioner) error {
	crbFound := &rbacv1.ClusterRoleBinding{}
	err := m.Client.Get(ctx, types.NamespacedName{Name: defaults.ClusterRoleBinding, Namespace: ""}, crbFound)
	if err != nil && errors.IsNotFound(err) {
		// Build ClusterRoleBinding using builder
		crb := builder.BuildClusterRoleBinding(nfsProvisioner)
		m.Log.Info("Creating ClusterRoleBinding", "clusterrolebinding.name", crb.Name)

		if err := m.Client.Create(ctx, crb); err != nil {
			m.Log.Error(err, "Failed to create ClusterRoleBinding", "clusterrolebinding.name", crb.Name)
			return err
		}
		m.Log.Info("Successfully created ClusterRoleBinding", "clusterrolebinding.name", crb.Name)
	} else if err != nil {
		m.Log.Error(err, "Failed to get ClusterRoleBinding")
		return err
	} else {
		m.Log.V(1).Info("ClusterRoleBinding already exists", "clusterrolebinding.name", crbFound.Name)
	}
	return nil
}

// ensureRole ensures the Role exists
func (m *RBACManager) ensureRole(ctx context.Context, nfsProvisioner *cachev1alpha1.NFSProvisioner) error {
	roleFound := &rbacv1.Role{}
	err := m.Client.Get(ctx, types.NamespacedName{Name: defaults.Role, Namespace: nfsProvisioner.Namespace}, roleFound)
	if err != nil && errors.IsNotFound(err) {
		// Build Role using builder
		role := builder.BuildRole(nfsProvisioner)

		// Set NFSProvisioner instance as the owner and controller
		if err := ctrl.SetControllerReference(nfsProvisioner, role, m.Scheme); err != nil {
			m.Log.Error(err, "Failed to set controller reference on Role")
			return err
		}

		m.Log.Info("Creating Role", "role.namespace", role.Namespace, "role.name", role.Name)
		if err = m.Client.Create(ctx, role); err != nil {
			m.Log.Error(err, "Failed to create Role", "role.namespace", role.Namespace, "role.name", role.Name)
			return err
		}
		m.Log.Info("Successfully created Role", "role.namespace", role.Namespace, "role.name", role.Name)
	} else if err != nil {
		m.Log.Error(err, "Failed to get Role")
		return err
	} else {
		m.Log.V(1).Info("Role already exists", "role.namespace", roleFound.Namespace, "role.name", roleFound.Name)
	}
	return nil
}

// ensureRoleBinding ensures the RoleBinding exists
func (m *RBACManager) ensureRoleBinding(ctx context.Context, nfsProvisioner *cachev1alpha1.NFSProvisioner) error {
	roleBindingFound := &rbacv1.RoleBinding{}
	err := m.Client.Get(ctx, types.NamespacedName{Name: defaults.RoleBinding, Namespace: nfsProvisioner.Namespace}, roleBindingFound)
	if err != nil && errors.IsNotFound(err) {
		// Build RoleBinding using builder
		roleBinding := builder.BuildRoleBinding(nfsProvisioner)

		// Set NFSProvisioner instance as the owner and controller
		if err := ctrl.SetControllerReference(nfsProvisioner, roleBinding, m.Scheme); err != nil {
			m.Log.Error(err, "Failed to set controller reference on RoleBinding")
			return err
		}

		m.Log.Info("Creating RoleBinding", "rolebinding.namespace", roleBinding.Namespace, "rolebinding.name", roleBinding.Name)
		if err = m.Client.Create(ctx, roleBinding); err != nil {
			m.Log.Error(err, "Failed to create RoleBinding", "rolebinding.namespace", roleBinding.Namespace, "rolebinding.name", roleBinding.Name)
			return err
		}
		m.Log.Info("Successfully created RoleBinding", "rolebinding.namespace", roleBinding.Namespace, "rolebinding.name", roleBinding.Name)
	} else if err != nil {
		m.Log.Error(err, "Failed to get RoleBinding")
		return err
	} else {
		m.Log.V(1).Info("RoleBinding already exists", "rolebinding.namespace", roleBindingFound.Namespace, "rolebinding.name", roleBindingFound.Name)
	}
	return nil
}
