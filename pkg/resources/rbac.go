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
	log := m.Log.WithValues("resource", m.GetResourceName())

	// Ensure ClusterRole
	if err := m.ensureClusterRole(ctx, nfsProvisioner, log); err != nil {
		return err
	}

	// Ensure ClusterRoleBinding
	if err := m.ensureClusterRoleBinding(ctx, nfsProvisioner, log); err != nil {
		return err
	}

	// Ensure Role
	if err := m.ensureRole(ctx, nfsProvisioner, log); err != nil {
		return err
	}

	// Ensure RoleBinding
	if err := m.ensureRoleBinding(ctx, nfsProvisioner, log); err != nil {
		return err
	}

	return nil
}

// ensureClusterRole ensures the ClusterRole exists
func (m *RBACManager) ensureClusterRole(ctx context.Context, nfsProvisioner *cachev1alpha1.NFSProvisioner, log interface{}) error {
	crFound := &rbacv1.ClusterRole{}
	err := m.Client.Get(ctx, types.NamespacedName{Name: defaults.ClusterRole, Namespace: ""}, crFound)
	if err != nil && errors.IsNotFound(err) {
		// Build ClusterRole using builder
		cr := builder.BuildClusterRole()
		m.Log.Info("Creating a new ClusterRole", "ClusterRole.Name", cr.Name)

		if err := m.Client.Create(ctx, cr); err != nil {
			m.Log.Error(err, "Failed to create a ClusterRole for NFSProvisioner", "ClusterRole.Name", cr.Name)
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}

// ensureClusterRoleBinding ensures the ClusterRoleBinding exists
func (m *RBACManager) ensureClusterRoleBinding(ctx context.Context, nfsProvisioner *cachev1alpha1.NFSProvisioner, log interface{}) error {
	crbFound := &rbacv1.ClusterRoleBinding{}
	err := m.Client.Get(ctx, types.NamespacedName{Name: defaults.ClusterRoleBinding, Namespace: ""}, crbFound)
	if err != nil && errors.IsNotFound(err) {
		// Build ClusterRoleBinding using builder
		crb := builder.BuildClusterRoleBinding(nfsProvisioner)
		m.Log.Info("Creating a new ClusterRoleBinding", "ClusterRoleBinding.Name", crb.Name)

		if err := m.Client.Create(ctx, crb); err != nil {
			m.Log.Error(err, "Failed to create a ClusterRoleBinding for NFSProvisioner", "ClusterRoleBinding.Name", crb.Name)
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}

// ensureRole ensures the Role exists
func (m *RBACManager) ensureRole(ctx context.Context, nfsProvisioner *cachev1alpha1.NFSProvisioner, log interface{}) error {
	roleFound := &rbacv1.Role{}
	err := m.Client.Get(ctx, types.NamespacedName{Name: defaults.Role, Namespace: nfsProvisioner.Namespace}, roleFound)
	if err != nil && errors.IsNotFound(err) {
		// Build Role using builder
		role := builder.BuildRole(nfsProvisioner)

		// Set NFSProvisioner instance as the owner and controller
		if err := ctrl.SetControllerReference(nfsProvisioner, role, m.Scheme); err != nil {
			return err
		}

		m.Log.Info("Creating a new Role", "Role.Namespace", role.Namespace, "Role.Name", role.Name)
		if err = m.Client.Create(ctx, role); err != nil {
			m.Log.Error(err, "Failed to create a Role for NFSProvisioner", "Role.Namespace", role.Namespace, "Role.Name", role.Name)
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}

// ensureRoleBinding ensures the RoleBinding exists
func (m *RBACManager) ensureRoleBinding(ctx context.Context, nfsProvisioner *cachev1alpha1.NFSProvisioner, log interface{}) error {
	roleBindingFound := &rbacv1.RoleBinding{}
	err := m.Client.Get(ctx, types.NamespacedName{Name: defaults.RoleBinding, Namespace: nfsProvisioner.Namespace}, roleBindingFound)
	if err != nil && errors.IsNotFound(err) {
		// Build RoleBinding using builder
		roleBinding := builder.BuildRoleBinding(nfsProvisioner)

		// Set NFSProvisioner instance as the owner and controller
		if err := ctrl.SetControllerReference(nfsProvisioner, roleBinding, m.Scheme); err != nil {
			return err
		}

		m.Log.Info("Creating a new RoleBinding", "RoleBinding.Namespace", roleBinding.Namespace, "RoleBinding.Name", roleBinding.Name)
		if err = m.Client.Create(ctx, roleBinding); err != nil {
			m.Log.Error(err, "Failed to create a RoleBinding for NFSProvisioner", "RoleBinding.Namespace", roleBinding.Namespace, "RoleBinding.Name", roleBinding.Name)
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}
