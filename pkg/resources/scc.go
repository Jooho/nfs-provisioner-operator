package resources

import (
	"context"
	"strings"

	securityv1 "github.com/openshift/api/security/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/pkg/builder"
	"github.com/jooho/nfs-provisioner-operator/pkg/defaults"
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

// isSCCCRDAvailable checks if SecurityContextConstraints CRD exists in the cluster
func (m *SCCManager) isSCCCRDAvailable(ctx context.Context) bool {
	crd := &apiextensionsv1.CustomResourceDefinition{}
	err := m.Client.Get(ctx, types.NamespacedName{Name: "securitycontextconstraints.security.openshift.io"}, crd)
	return err == nil
}

// EnsureResource ensures the SCC exists and is properly configured
func (m *SCCManager) EnsureResource(ctx context.Context, nfsProvisioner *cachev1alpha1.NFSProvisioner) error {
	log := m.Log.WithValues("resource", m.GetResourceName())

	// Check if SecurityContextConstraints CRD is available in the cluster
	if !m.isSCCCRDAvailable(ctx) {
		log.Info("SecurityContextConstraints CRD is not available in cluster, skipping SCC creation")
		return nil
	}

	sccFound := &securityv1.SecurityContextConstraints{}
	err := m.Client.Get(ctx, types.NamespacedName{Name: defaults.SecurityContextConstraints, Namespace: ""}, sccFound)
	if err != nil && errors.IsNotFound(err) {
		// Build SCC using builder
		scc := builder.BuildSCC(nfsProvisioner)
		log.Info("Creating a new SecurityContextConstraints", "SecurityContextConstraints.Name", scc.Name)

		if err := m.Client.Create(ctx, scc); err != nil {
			log.Error(err, "Failed to create a new SecurityContextConstraints", "SecurityContextConstraints.Name", scc.Name)
			return err
		}
		return nil
	} else if err != nil {
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
