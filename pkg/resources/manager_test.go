package resources

import (
	"testing"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	securityv1 "github.com/openshift/api/security/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/pkg/defaults"
)

func TestResourceManagers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Resource Manager Suite")
}

var _ = Describe("Resource Manager Set", func() {
	var (
		testClient         client.Client
		testScheme         *runtime.Scheme
		nfsProvisioner     *cachev1alpha1.NFSProvisioner
		resourceManagerSet *ResourceManagerSet
	)

	BeforeEach(func(ctx SpecContext) {
		// Create scheme and add types
		testScheme = runtime.NewScheme()
		Expect(corev1.AddToScheme(testScheme)).To(Succeed())
		Expect(appsv1.AddToScheme(testScheme)).To(Succeed())
		Expect(rbacv1.AddToScheme(testScheme)).To(Succeed())
		Expect(storagev1.AddToScheme(testScheme)).To(Succeed())
		Expect(cachev1alpha1.AddToScheme(testScheme)).To(Succeed())
		Expect(securityv1.AddToScheme(testScheme)).To(Succeed())
		Expect(apiextensionsv1.AddToScheme(testScheme)).To(Succeed())

		// Create SecurityContextConstraints CRD for testing
		sccCRD := &apiextensionsv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: "securitycontextconstraints.security.openshift.io",
			},
			Spec: apiextensionsv1.CustomResourceDefinitionSpec{
				Group: "security.openshift.io",
				Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
					{
						Name:    "v1",
						Served:  true,
						Storage: true,
						Schema: &apiextensionsv1.CustomResourceValidation{
							OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
								Type: "object",
							},
						},
					},
				},
				Scope: apiextensionsv1.ClusterScoped,
				Names: apiextensionsv1.CustomResourceDefinitionNames{
					Plural: "securitycontextconstraints",
					Kind:   "SecurityContextConstraints",
				},
			},
		}

		// Create fake client with the CRD
		testClient = fake.NewClientBuilder().WithScheme(testScheme).WithObjects(sccCRD).Build()

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
		resourceManagerSet = NewResourceManagerSet(testClient, logr.Discard(), testScheme)
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
		It("should create all resource managers", func(ctx SpecContext) {
			Expect(resourceManagerSet.SCC.GetResourceName()).To(Equal("SecurityContextConstraints"))
			Expect(resourceManagerSet.PVC.GetResourceName()).To(Equal("PersistentVolumeClaim"))
			Expect(resourceManagerSet.ServiceAccount.GetResourceName()).To(Equal("ServiceAccount"))
			Expect(resourceManagerSet.RBAC.GetResourceName()).To(Equal("RBAC"))
			Expect(resourceManagerSet.Deployment.GetResourceName()).To(Equal("Deployment"))
			Expect(resourceManagerSet.Service.GetResourceName()).To(Equal("Service"))
			Expect(resourceManagerSet.StorageClass.GetResourceName()).To(Equal("StorageClass"))
		})

		It("should return managed resource names", func(ctx SpecContext) {
			names := resourceManagerSet.GetManagedResourceNames()
			Expect(names).To(ContainElements("SecurityContextConstraints", "PersistentVolumeClaim", "ServiceAccount", "RBAC", "Deployment", "Service", "StorageClass"))
		})

		It("should ensure all resources successfully", func(ctx SpecContext) {
			err := resourceManagerSet.EnsureAllResources(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should be idempotent (calling EnsureAllResources twice)", func(ctx SpecContext) {
			// First call
			err := resourceManagerSet.EnsureAllResources(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Second call - should not error
			err = resourceManagerSet.EnsureAllResources(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("SCCManager", func() {
		var sccManager *SCCManager

		BeforeEach(func(ctx SpecContext) {
			baseManager := NewBaseResourceManager(testClient, logr.Discard(), testScheme)
			sccManager = NewSCCManager(baseManager)
		})

		It("should create SCC when it doesn't exist", func(ctx SpecContext) {
			err := sccManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify SCC was created
			scc := &securityv1.SecurityContextConstraints{}
			err = testClient.Get(ctx, types.NamespacedName{Name: defaults.SecurityContextConstraints}, scc)
			Expect(err).NotTo(HaveOccurred())
			Expect(scc.Users).To(ContainElement("system:serviceaccount:test-namespace:" + defaults.ServiceAccount))
		})

		It("should add user to existing SCC", func(ctx SpecContext) {
			// Create existing SCC without our user
			existingSCC := &securityv1.SecurityContextConstraints{
				ObjectMeta: metav1.ObjectMeta{
					Name: defaults.SecurityContextConstraints,
				},
				Users: []string{"system:serviceaccount:other-namespace:other-sa"},
			}
			Expect(testClient.Create(ctx, existingSCC)).To(Succeed())

			err := sccManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify user was added
			scc := &securityv1.SecurityContextConstraints{}
			err = testClient.Get(ctx, types.NamespacedName{Name: defaults.SecurityContextConstraints}, scc)
			Expect(err).NotTo(HaveOccurred())
			Expect(scc.Users).To(ContainElement("system:serviceaccount:test-namespace:" + defaults.ServiceAccount))
			Expect(scc.Users).To(ContainElement("system:serviceaccount:other-namespace:other-sa"))
		})
	})

	Describe("PVCManager", func() {
		var pvcManager *PVCManager

		BeforeEach(func(ctx SpecContext) {
			baseManager := NewBaseResourceManager(testClient, logr.Discard(), testScheme)
			pvcManager = NewPVCManager(baseManager)
		})

		It("should create PVC when using PVC storage", func(ctx SpecContext) {
			nfsProvisioner.Spec.SCForNFSPvc = "local-storage"
			nfsProvisioner.Spec.HostPathDir = ""

			err := pvcManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify PVC was created
			pvc := &corev1.PersistentVolumeClaim{}
			err = testClient.Get(ctx, types.NamespacedName{Name: defaults.Pvc, Namespace: nfsProvisioner.Namespace}, pvc)
			Expect(err).NotTo(HaveOccurred())
			Expect(pvc.Spec.AccessModes).To(ContainElement(corev1.ReadWriteOnce))
		})

		It("should skip PVC creation when using hostPath", func(ctx SpecContext) {
			nfsProvisioner.Spec.HostPathDir = "/tmp/nfs"
			err := pvcManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify PVC was not created
			pvc := &corev1.PersistentVolumeClaim{}
			err = testClient.Get(ctx, types.NamespacedName{Name: defaults.Pvc, Namespace: nfsProvisioner.Namespace}, pvc)
			Expect(err).To(HaveOccurred())
		})

		It("should use existing PVC when specified", func(ctx SpecContext) {
			nfsProvisioner.Spec.Pvc = "existing-pvc"

			// Create existing PVC
			existingPVC := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-pvc",
					Namespace: nfsProvisioner.Namespace,
				},
			}
			Expect(testClient.Create(ctx, existingPVC)).To(Succeed())

			err := pvcManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("ServiceAccountManager", func() {
		var saManager *ServiceAccountManager

		BeforeEach(func(ctx SpecContext) {
			baseManager := NewBaseResourceManager(testClient, logr.Discard(), testScheme)
			saManager = NewServiceAccountManager(baseManager)
		})

		It("should create ServiceAccount when it doesn't exist", func(ctx SpecContext) {
			err := saManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify ServiceAccount was created
			sa := &corev1.ServiceAccount{}
			err = testClient.Get(ctx, types.NamespacedName{Name: defaults.ServiceAccount, Namespace: nfsProvisioner.Namespace}, sa)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should not recreate existing ServiceAccount", func(ctx SpecContext) {
			// Create existing ServiceAccount
			existingSA := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaults.ServiceAccount,
					Namespace: nfsProvisioner.Namespace,
				},
			}
			Expect(testClient.Create(ctx, existingSA)).To(Succeed())

			err := saManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify ServiceAccount still exists
			sa := &corev1.ServiceAccount{}
			err = testClient.Get(ctx, types.NamespacedName{Name: defaults.ServiceAccount, Namespace: nfsProvisioner.Namespace}, sa)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
