package resources

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
)

// ResourceManager defines the interface for managing Kubernetes resources
type ResourceManager interface {
	// EnsureResource ensures the resource exists and is in the desired state
	EnsureResource(ctx context.Context, nfsProvisioner *cachev1alpha1.NFSProvisioner) error

	// GetResourceName returns the name of the resource this manager handles
	GetResourceName() string
}

// BaseResourceManager provides common functionality for all resource managers
type BaseResourceManager struct {
	Client client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// NewBaseResourceManager creates a new BaseResourceManager
func NewBaseResourceManager(client client.Client, log logr.Logger, scheme *runtime.Scheme) BaseResourceManager {
	return BaseResourceManager{
		Client: client,
		Log:    log,
		Scheme: scheme,
	}
}
