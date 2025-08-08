package resources

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/controllers/defaults"
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
	log := m.Log.WithValues("resource", m.GetResourceName())

	// Check if the service already exists
	svcFound := &corev1.Service{}
	err := m.Client.Get(ctx, types.NamespacedName{Name: defaults.Service, Namespace: nfsProvisioner.Namespace}, svcFound)
	if err != nil && errors.IsNotFound(err) {
		// Define a new service
		svc := m.buildService(nfsProvisioner)

		log.Info("Creating a new Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)

		if err = m.Client.Create(ctx, svc); err != nil {
			log.Error(err, "Failed to create a Service for NFSProvisioner", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}

// buildService creates a new Service object
func (m *ServiceManager) buildService(nfsProvisioner *cachev1alpha1.NFSProvisioner) *corev1.Service {
	ls := labelsForNFSProvisioner(nfsProvisioner.Name)
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaults.Service,
			Namespace: nfsProvisioner.Namespace,
			Labels:    ls,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{Name: "nfs",
					Port: 2049},
				{Name: "nfs-udp",
					Port:     2049,
					Protocol: "UDP"},
				{Name: "nlockmgr",
					Port: 32803},
				{Name: "nlockmgr-udp",
					Port:     32803,
					Protocol: "UDP"},
				{Name: "mountd",
					Port: 20048},
				{Name: "mountd-udp",
					Port:     20048,
					Protocol: "UDP"},
				{Name: "rquotad",
					Port: 875},
				{Name: "rquotad-udp",
					Port:     875,
					Protocol: "UDP"},
				{Name: "rpcbind",
					Port: 111},
				{Name: "rpcbind-udp",
					Port:     111,
					Protocol: "UDP"},
				{Name: "statd",
					Port: 662},
				{Name: "statd-udp",
					Port:     662,
					Protocol: "UDP"},
			},
			Selector: ls,
		},
	}

	ctrl.SetControllerReference(nfsProvisioner, svc, m.Scheme)
	return svc
}
