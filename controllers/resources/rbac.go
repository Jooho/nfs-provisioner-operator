package resources

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/controllers/defaults"
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
	if err != nil {
		if errors.IsNotFound(err) {
			cr := m.buildClusterRole(nfsProvisioner)
			m.Log.Info("Creating a new ClusterRole", "ClusterRole.Name", cr.Name)

			if err := m.Client.Create(ctx, cr); err != nil {
				m.Log.Error(err, "Failed to create a ClusterRole for NFSProvisioner", "ClusterRole.Namespace", cr.Namespace, "ClusterRole.Name", cr.Name)
				return err
			}
		} else {
			return err
		}
	}
	return nil
}

// ensureClusterRoleBinding ensures the ClusterRoleBinding exists
func (m *RBACManager) ensureClusterRoleBinding(ctx context.Context, nfsProvisioner *cachev1alpha1.NFSProvisioner, log interface{}) error {
	crbFound := &rbacv1.ClusterRoleBinding{}
	err := m.Client.Get(ctx, types.NamespacedName{Name: defaults.ClusterRoleBinding, Namespace: ""}, crbFound)
	if err != nil {
		if errors.IsNotFound(err) {
			crb := m.buildClusterRoleBinding(nfsProvisioner)
			m.Log.Info("Creating a new ClusterRoleBinding", "ClusterRoleBinding.Name", crb.Name)

			if err := m.Client.Create(ctx, crb); err != nil {
				m.Log.Error(err, "Failed to create a ClusterRoleBinding for NFSProvisioner", "ClusterRoleBinding.Name", crb.Name)
				return err
			}
		} else {
			return err
		}
	}
	return nil
}

// ensureRole ensures the Role exists
func (m *RBACManager) ensureRole(ctx context.Context, nfsProvisioner *cachev1alpha1.NFSProvisioner, log interface{}) error {
	roleFound := &rbacv1.Role{}
	err := m.Client.Get(ctx, types.NamespacedName{Name: defaults.Role, Namespace: nfsProvisioner.Namespace}, roleFound)
	if err != nil {
		if errors.IsNotFound(err) {
			role := m.buildRole(nfsProvisioner)
			m.Log.Info("Creating a new Role", "Role.Namespace", role.Namespace, "Role.Name", role.Name)
			if err := m.Client.Create(ctx, role); err != nil {
				m.Log.Error(err, "Failed to create a Role for NFSProvisioner", "Role.Namespace", role.Namespace, "Role.Name", role.Name)
				return err
			}
		} else {
			return err
		}
	}
	return nil
}

// ensureRoleBinding ensures the RoleBinding exists
func (m *RBACManager) ensureRoleBinding(ctx context.Context, nfsProvisioner *cachev1alpha1.NFSProvisioner, log interface{}) error {
	roleBindingFound := &rbacv1.RoleBinding{}
	err := m.Client.Get(ctx, types.NamespacedName{Name: defaults.RoleBinding, Namespace: nfsProvisioner.Namespace}, roleBindingFound)
	if err != nil {
		if errors.IsNotFound(err) {
			roleBinding := m.buildRoleBinding(nfsProvisioner)
			m.Log.Info("Creating a new RoleBinding", "RoleBinding.Namespace", roleBinding.Namespace, "RoleBinding.Name", roleBinding.Name)

			if err := m.Client.Create(ctx, roleBinding); err != nil {
				m.Log.Error(err, "Failed to create a RoleBinding for NFSProvisioner", "RoleBinding.Namespace", roleBinding.Namespace, "RoleBinding.Name", roleBinding.Name)
				return err
			}
		} else {
			return err
		}
	}
	return nil
}

// buildClusterRole creates a new ClusterRole object
func (m *RBACManager) buildClusterRole(nfsProvisioner *cachev1alpha1.NFSProvisioner) *rbacv1.ClusterRole {
	cr := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaults.ClusterRole,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"persistentvolumes"},
				Verbs:     []string{"get", "list", "watch", "create", "delete"},
			}, {
				APIGroups: []string{""},
				Resources: []string{"persistentvolumeclaims"},
				Verbs:     []string{"get", "list", "watch", "update"},
			}, {
				APIGroups: []string{"storage.k8s.io"},
				Resources: []string{"storageclasses"},
				Verbs:     []string{"get", "list", "watch"},
			}, {
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs:     []string{"create", "update", "patch"},
			}, {
				APIGroups:     []string{"policy"},
				Resources:     []string{"podsecuritypolicies"},
				ResourceNames: []string{"nfs-provisioner"},
				Verbs:         []string{"use"},
			},
		},
	}

	ctrl.SetControllerReference(nfsProvisioner, cr, m.Scheme)
	return cr
}

// buildClusterRoleBinding creates a new ClusterRoleBinding object
func (m *RBACManager) buildClusterRoleBinding(nfsProvisioner *cachev1alpha1.NFSProvisioner) *rbacv1.ClusterRoleBinding {
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaults.ClusterRoleBinding,
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      defaults.ServiceAccount,
			Namespace: nfsProvisioner.Namespace,
		}},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     defaults.ClusterRole,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	ctrl.SetControllerReference(nfsProvisioner, crb, m.Scheme)
	return crb
}

// buildRole creates a new Role object
func (m *RBACManager) buildRole(nfsProvisioner *cachev1alpha1.NFSProvisioner) *rbacv1.Role {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaults.Role,
			Namespace: nfsProvisioner.Namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"endpoints"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "delete"},
			}, {
				APIGroups: []string{""},
				Resources: []string{"services"},
				Verbs:     []string{"get"},
			},
		},
	}

	ctrl.SetControllerReference(nfsProvisioner, role, m.Scheme)
	return role
}

// buildRoleBinding creates a new RoleBinding object
func (m *RBACManager) buildRoleBinding(nfsProvisioner *cachev1alpha1.NFSProvisioner) *rbacv1.RoleBinding {
	rolebinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaults.RoleBinding,
			Namespace: nfsProvisioner.Namespace,
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      defaults.ServiceAccount,
			Namespace: nfsProvisioner.Namespace,
		}},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			Name:     defaults.Role,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	ctrl.SetControllerReference(nfsProvisioner, rolebinding, m.Scheme)
	return rolebinding
}
