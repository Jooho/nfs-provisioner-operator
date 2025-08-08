package resources

import (
	"context"
	"strings"

	securityv1 "github.com/openshift/api/security/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/controllers/defaults"
)

// SCCManager manages SecurityContextConstraints resources
type SCCManager struct {
	BaseResourceManager
}

// NewSCCManager creates a new SCCManager
func NewSCCManager(base BaseResourceManager) *SCCManager {
	return &SCCManager{
		BaseResourceManager: base,
	}
}

// GetResourceName returns the name of the resource this manager handles
func (m *SCCManager) GetResourceName() string {
	return "SecurityContextConstraints"
}

// EnsureResource ensures the SCC exists and is properly configured
func (m *SCCManager) EnsureResource(ctx context.Context, nfsProvisioner *cachev1alpha1.NFSProvisioner) error {
	log := m.Log.WithValues("resource", m.GetResourceName())

	sccFound := &securityv1.SecurityContextConstraints{}
	err := m.Client.Get(ctx, types.NamespacedName{Name: defaults.SecurityContextContrants, Namespace: ""}, sccFound)

	if err != nil {
		if errors.IsNotFound(err) {
			// Create new SCC
			scc := m.buildSCC(nfsProvisioner)
			log.Info("Creating a new SecurityContextConstraints", "SecurityContextConstraints.Name", scc.Name)

			if err := m.Client.Create(ctx, scc); err != nil {
				log.Error(err, "Failed to create a new SecurityContextConstraints", "SecurityContextConstraints.Name", scc.Name)
				return err
			}
			return nil
		}
		return err
	}

	// Update existing SCC - add namespace user if not present
	userToAdd := "system:serviceaccount:" + nfsProvisioner.Namespace + ":" + defaults.ServiceAccount
	userExists := false

	for _, user := range sccFound.Users {
		if strings.Contains(user, nfsProvisioner.Namespace+":"+defaults.ServiceAccount) {
			userExists = true
			break
		}
	}

	if !userExists {
		sccFound.Users = append(sccFound.Users, userToAdd)
		log.Info("Adding user to existing SecurityContextConstraints", "user", userToAdd)

		if err := m.Client.Update(ctx, sccFound); err != nil {
			log.Error(err, "Failed to update SecurityContextConstraints", "SecurityContextConstraints.Name", sccFound.Name)
			return err
		}
	}

	return nil
}

// buildSCC creates a new SecurityContextConstraints object
func (m *SCCManager) buildSCC(nfsProvisioner *cachev1alpha1.NFSProvisioner) *securityv1.SecurityContextConstraints {
	scc := &securityv1.SecurityContextConstraints{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaults.SecurityContextContrants,
		},
		AllowHostDirVolumePlugin: true,
		AllowHostIPC:             false,
		AllowHostNetwork:         false,
		AllowHostPID:             false,
		AllowHostPorts:           false,
		AllowPrivilegedContainer: false,
		AllowedCapabilities:      []corev1.Capability{"DAC_READ_SEARCH", "SYS_RESOURCE"},
		DefaultAddCapabilities:   nil,
		Priority:                 nil,
		ReadOnlyRootFilesystem:   false,
		RequiredDropCapabilities: []corev1.Capability{"KILL", "MKNOD", "SYS_CHROOT"},
		RunAsUser: securityv1.RunAsUserStrategyOptions{
			Type: securityv1.RunAsUserStrategyRunAsAny,
		},
		SELinuxContext: securityv1.SELinuxContextStrategyOptions{
			Type: securityv1.SELinuxStrategyMustRunAs,
		},
		Users: []string{"system:serviceaccount:" + nfsProvisioner.Namespace + ":" + defaults.ServiceAccount},
		SupplementalGroups: securityv1.SupplementalGroupsStrategyOptions{
			Type: securityv1.SupplementalGroupsStrategyRunAsAny,
		},
		Volumes: []securityv1.FSType{
			securityv1.FSTypeConfigMap,
			securityv1.FSTypeDownwardAPI,
			securityv1.FSTypeEmptyDir,
			securityv1.FSTypePersistentVolumeClaim,
			securityv1.FSTypeHostPath,
			securityv1.FSTypeSecret,
		},
		FSGroup: securityv1.FSGroupStrategyOptions{
			Type: securityv1.FSGroupStrategyMustRunAs,
		},
	}

	// Set controller reference
	ctrl.SetControllerReference(nfsProvisioner, scc, m.Scheme)
	return scc
}
