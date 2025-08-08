package resources

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	securityv1 "github.com/openshift/api/security/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/controllers/defaults"
)

func TestResourceManagers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Resource Manager Suite")
}

var _ = Describe("Resource Manager Set", func() {
	var (
		ctx                context.Context
		client             client.Client
		nfsProvisioner     *cachev1alpha1.NFSProvisioner
		resourceManagerSet *ResourceManagerSet
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Create scheme and add types
		scheme := runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		Expect(appsv1.AddToScheme(scheme)).To(Succeed())
		Expect(rbacv1.AddToScheme(scheme)).To(Succeed())
		Expect(storagev1.AddToScheme(scheme)).To(Succeed())
		Expect(cachev1alpha1.AddToScheme(scheme)).To(Succeed())
		Expect(securityv1.AddToScheme(scheme)).To(Succeed())

		// Create fake client
		client = fake.NewClientBuilder().WithScheme(scheme).Build()

		// Create test NFSProvisioner instance
		nfsProvisioner = &cachev1alpha1.NFSProvisioner{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-nfs",
				Namespace: "test-namespace",
				UID:       "test-uid",
			},
			Spec: cachev1alpha1.NFSProvisionerSpec{
				StorageSize: "10Gi",
			},
		}

		// Create resource manager set
		resourceManagerSet = NewResourceManagerSet(client, logr.Discard(), scheme)
		Expect(resourceManagerSet).NotTo(BeNil())
		Expect(resourceManagerSet.SCC).NotTo(BeNil())
		Expect(resourceManagerSet.PVC).NotTo(BeNil())
		Expect(resourceManagerSet.ServiceAccount).NotTo(BeNil())
		Expect(resourceManagerSet.RBAC).NotTo(BeNil())
		Expect(resourceManagerSet.Deployment).NotTo(BeNil())
		Expect(resourceManagerSet.Service).NotTo(BeNil())
		Expect(resourceManagerSet.StorageClass).NotTo(BeNil())
	})

	Describe("ResourceManagerSet", func() {
		It("should create all resource managers", func() {
			Expect(resourceManagerSet.SCC.GetResourceName()).To(Equal("SecurityContextConstraints"))
			Expect(resourceManagerSet.PVC.GetResourceName()).To(Equal("PersistentVolumeClaim"))
			Expect(resourceManagerSet.ServiceAccount.GetResourceName()).To(Equal("ServiceAccount"))
			Expect(resourceManagerSet.RBAC.GetResourceName()).To(Equal("RBAC"))
			Expect(resourceManagerSet.Deployment.GetResourceName()).To(Equal("Deployment"))
			Expect(resourceManagerSet.Service.GetResourceName()).To(Equal("Service"))
			Expect(resourceManagerSet.StorageClass.GetResourceName()).To(Equal("StorageClass"))
		})

		It("should return managed resource names", func() {
			names := resourceManagerSet.GetManagedResourceNames()
			Expect(names).To(ContainElements("SecurityContextConstraints", "PersistentVolumeClaim", "ServiceAccount", "RBAC", "Deployment", "Service", "StorageClass"))
		})

		It("should ensure all resources successfully", func() {
			err := resourceManagerSet.EnsureAllResources(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("SCCManager", func() {
		var sccManager *SCCManager

		BeforeEach(func() {
			baseManager := NewBaseResourceManager(client, logr.Discard(), scheme.Scheme)
			sccManager = NewSCCManager(baseManager)
		})

		It("should create SCC when it doesn't exist", func() {
			err := sccManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify SCC was created
			scc := &securityv1.SecurityContextConstraints{}
			err = client.Get(ctx, types.NamespacedName{Name: defaults.SecurityContextContrants}, scc)
			Expect(err).NotTo(HaveOccurred())
			Expect(scc.Users).To(ContainElement("system:serviceaccount:test-namespace:" + defaults.ServiceAccount))
		})

		It("should add user to existing SCC", func() {
			// Create existing SCC without our user
			existingSCC := &securityv1.SecurityContextConstraints{
				ObjectMeta: metav1.ObjectMeta{
					Name: defaults.SecurityContextContrants,
				},
				Users: []string{"system:serviceaccount:other-namespace:other-sa"},
			}
			Expect(client.Create(ctx, existingSCC)).To(Succeed())

			err := sccManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify user was added
			scc := &securityv1.SecurityContextConstraints{}
			err = client.Get(ctx, types.NamespacedName{Name: defaults.SecurityContextContrants}, scc)
			Expect(err).NotTo(HaveOccurred())
			Expect(scc.Users).To(ContainElement("system:serviceaccount:test-namespace:" + defaults.ServiceAccount))
			Expect(scc.Users).To(ContainElement("system:serviceaccount:other-namespace:other-sa"))
		})
	})

	Describe("PVCManager", func() {
		var pvcManager *PVCManager

		BeforeEach(func() {
			baseManager := NewBaseResourceManager(client, logr.Discard(), scheme.Scheme)
			pvcManager = NewPVCManager(baseManager)
		})

		It("should create PVC when using PVC storage", func() {
			err := pvcManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify PVC was created
			pvc := &corev1.PersistentVolumeClaim{}
			err = client.Get(ctx, types.NamespacedName{Name: defaults.Pvc, Namespace: nfsProvisioner.Namespace}, pvc)
			Expect(err).NotTo(HaveOccurred())
			Expect(pvc.Spec.AccessModes).To(ContainElement(corev1.ReadWriteOnce))
		})

		It("should skip PVC creation when using hostPath", func() {
			nfsProvisioner.Spec.HostPathDir = "/tmp/nfs"
			err := pvcManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify PVC was not created
			pvc := &corev1.PersistentVolumeClaim{}
			err = client.Get(ctx, types.NamespacedName{Name: defaults.Pvc, Namespace: nfsProvisioner.Namespace}, pvc)
			Expect(err).To(HaveOccurred())
		})

		It("should use existing PVC when specified", func() {
			nfsProvisioner.Spec.Pvc = "existing-pvc"

			// Create existing PVC
			existingPVC := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-pvc",
					Namespace: nfsProvisioner.Namespace,
				},
			}
			Expect(client.Create(ctx, existingPVC)).To(Succeed())

			err := pvcManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("ServiceAccountManager", func() {
		var saManager *ServiceAccountManager

		BeforeEach(func() {
			baseManager := NewBaseResourceManager(client, logr.Discard(), scheme.Scheme)
			saManager = NewServiceAccountManager(baseManager)
		})

		It("should create ServiceAccount when it doesn't exist", func() {
			err := saManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify ServiceAccount was created
			sa := &corev1.ServiceAccount{}
			err = client.Get(ctx, types.NamespacedName{Name: defaults.ServiceAccount, Namespace: nfsProvisioner.Namespace}, sa)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should not recreate existing ServiceAccount", func() {
			// Create existing ServiceAccount
			existingSA := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaults.ServiceAccount,
					Namespace: nfsProvisioner.Namespace,
				},
			}
			Expect(client.Create(ctx, existingSA)).To(Succeed())

			err := saManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify ServiceAccount still exists
			sa := &corev1.ServiceAccount{}
			err = client.Get(ctx, types.NamespacedName{Name: defaults.ServiceAccount, Namespace: nfsProvisioner.Namespace}, sa)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
