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

// ServiceManager manages Service resources
type ServiceManager struct {
	BaseResourceManager
}

// NewServiceManager creates a new ServiceManager
func NewServiceManager(base BaseResourceManager) *ServiceManager {
	return &ServiceManager{
		BaseResourceManager: base,
	}
}

// GetResourceName returns the name of the resource this manager handles
func (m *ServiceManager) GetResourceName() string {
	return "Service"
}

// EnsureResource ensures the Service exists
func (m *ServiceManager) EnsureResource(ctx context.Context, nfsProvisioner *cachev1alpha1.NFSProvisioner) error {
	log := m.Log.WithValues(
		"resource", m.GetResourceName(),
		"nfsprovisioner.name", nfsProvisioner.Name,
		"nfsprovisioner.namespace", nfsProvisioner.Namespace,
	)

	// Check if the service already exists
	svcFound := &corev1.Service{}
	err := m.Client.Get(ctx, types.NamespacedName{Name: defaults.Service, Namespace: nfsProvisioner.Namespace}, svcFound)
	if err != nil && errors.IsNotFound(err) {
		// Build service using builder
		svc := builder.BuildService(nfsProvisioner)

		// Set NFSProvisioner instance as the owner and controller
		if err := ctrl.SetControllerReference(nfsProvisioner, svc, m.Scheme); err != nil {
			log.Error(err, "Failed to set controller reference on Service")
			return err
		}

		log.Info("Creating Service", "service.namespace", svc.Namespace, "service.name", svc.Name)
		if err = m.Client.Create(ctx, svc); err != nil {
			log.Error(err, "Failed to create Service", "service.namespace", svc.Namespace, "service.name", svc.Name)
			return err
		}
		log.Info("Successfully created Service", "service.namespace", svc.Namespace, "service.name", svc.Name)
	} else if err != nil {
		log.Error(err, "Failed to get Service")
		return err
	} else {
		log.V(1).Info("Service already exists", "service.namespace", svcFound.Namespace, "service.name", svcFound.Name)
	}

	return nil
}
