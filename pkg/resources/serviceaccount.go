package resources

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/pkg/builder"
	"github.com/jooho/nfs-provisioner-operator/pkg/defaults"
)

// ServiceAccountManager manages ServiceAccount resources
type ServiceAccountManager struct {
	BaseResourceManager
}

// NewServiceAccountManager creates a new ServiceAccountManager
func NewServiceAccountManager(base BaseResourceManager) *ServiceAccountManager {
	return &ServiceAccountManager{
		BaseResourceManager: base,
	}
}

// GetResourceName returns the name of the resource this manager handles
func (m *ServiceAccountManager) GetResourceName() string {
	return "ServiceAccount"
}

// EnsureResource ensures the ServiceAccount exists
func (m *ServiceAccountManager) EnsureResource(ctx context.Context, nfsProvisioner *cachev1alpha1.NFSProvisioner) error {
	log := m.Log.WithValues("resource", m.GetResourceName())

	// Check if ServiceAccount already exists
	saFound := &corev1.ServiceAccount{}
	err := m.Client.Get(ctx, types.NamespacedName{Name: defaults.ServiceAccount, Namespace: nfsProvisioner.Namespace}, saFound)
	if err != nil && errors.IsNotFound(err) {
		// Build ServiceAccount using builder
		sa := builder.BuildServiceAccount(nfsProvisioner)

		// Set NFSProvisioner instance as the owner and controller
		if err := ctrl.SetControllerReference(nfsProvisioner, sa, m.Scheme); err != nil {
			return err
		}

		log.Info("Creating a new ServiceAccount", "ServiceAccount.Namespace", sa.Namespace, "ServiceAccount.Name", sa.Name)
		if err = m.Client.Create(ctx, sa); err != nil {
			log.Error(err, "Failed to create a new ServiceAccount", "ServiceAccount.Namespace", sa.Namespace, "ServiceAccount.Name", sa.Name)
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}
