package resources

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/pkg/builder"
	"github.com/jooho/nfs-provisioner-operator/pkg/defaults"
)

// DeploymentManager manages Deployment resources
type DeploymentManager struct {
	BaseResourceManager
}

// NewDeploymentManager creates a new DeploymentManager
func NewDeploymentManager(base BaseResourceManager) *DeploymentManager {
	return &DeploymentManager{
		BaseResourceManager: base,
	}
}

// GetResourceName returns the name of the resource this manager handles
func (m *DeploymentManager) GetResourceName() string {
	return "Deployment"
}

// EnsureResource ensures the Deployment exists
func (m *DeploymentManager) EnsureResource(ctx context.Context, nfsProvisioner *cachev1alpha1.NFSProvisioner) error {
	log := m.Log.WithValues(
		"resource", m.GetResourceName(),
		"nfsprovisioner.name", nfsProvisioner.Name,
		"nfsprovisioner.namespace", nfsProvisioner.Namespace,
	)

	// Check if the deployment already exists
	deployFound := &appsv1.Deployment{}
	err := m.Client.Get(ctx, types.NamespacedName{Name: defaults.Deployment, Namespace: nfsProvisioner.Namespace}, deployFound)
	if err != nil && errors.IsNotFound(err) {
		// Build deployment using builder
		dep := builder.BuildDeployment(nfsProvisioner)

		// Set NFSProvisioner instance as the owner and controller
		if refErr := ctrl.SetControllerReference(nfsProvisioner, dep, m.Scheme); refErr != nil {
			return fmt.Errorf("failed to set controller reference on Deployment: %w", refErr)
		}

		log.Info("Creating Deployment", "deployment.namespace", dep.Namespace, "deployment.name", dep.Name)
		if err = m.Client.Create(ctx, dep); err != nil {
			log.Error(err, "Failed to create Deployment", "deployment.namespace", dep.Namespace, "deployment.name", dep.Name)
			return err
		}
		log.Info("Successfully created Deployment", "deployment.namespace", dep.Namespace, "deployment.name", dep.Name)
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		return err
	} else {
		log.V(1).Info("Deployment already exists", "deployment.namespace", deployFound.Namespace, "deployment.name", deployFound.Name)
	}

	return nil
}
