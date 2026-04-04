package resources

import (
	"context"
	"fmt"

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
	log := m.Log.WithValues(
		"resource", m.GetResourceName(),
		"nfsprovisioner.name", nfsProvisioner.Name,
		"nfsprovisioner.namespace", nfsProvisioner.Namespace,
	)

	// Check if ServiceAccount already exists
	saFound := &corev1.ServiceAccount{}
	err := m.Client.Get(ctx, types.NamespacedName{Name: defaults.ServiceAccount, Namespace: nfsProvisioner.Namespace}, saFound)
	if err != nil && errors.IsNotFound(err) {
		// Build ServiceAccount using builder
		sa := builder.BuildServiceAccount(nfsProvisioner)

		// Set NFSProvisioner instance as the owner and controller
		if refErr := ctrl.SetControllerReference(nfsProvisioner, sa, m.Scheme); refErr != nil {
			return fmt.Errorf("failed to set controller reference on ServiceAccount: %w", refErr)
		}

		log.Info("Creating ServiceAccount", "serviceaccount.namespace", sa.Namespace, "serviceaccount.name", sa.Name)
		if err = m.Client.Create(ctx, sa); err != nil {
			log.Error(err, "Failed to create ServiceAccount", "serviceaccount.namespace", sa.Namespace, "serviceaccount.name", sa.Name)
			return err
		}
		log.Info("Successfully created ServiceAccount", "serviceaccount.namespace", sa.Namespace, "serviceaccount.name", sa.Name)
	} else if err != nil {
		log.Error(err, "Failed to get ServiceAccount")
		return err
	} else {
		log.V(1).Info("ServiceAccount already exists", "serviceaccount.namespace", saFound.Namespace, "serviceaccount.name", saFound.Name)
	}

	return nil
}
