package resources

import (
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/pkg/defaults"
)

var _ = Describe("StorageClassManager", func() {
	var (
		storageClassManager *StorageClassManager
		nfsProvisioner      *cachev1alpha1.NFSProvisioner
		testScheme          *runtime.Scheme
	)

	BeforeEach(func(ctx SpecContext) {
		// Create scheme
		testScheme = runtime.NewScheme()
		Expect(storagev1.AddToScheme(testScheme)).To(Succeed())
		Expect(cachev1alpha1.AddToScheme(testScheme)).To(Succeed())

		// Create test NFSProvisioner instance
		nfsProvisioner = &cachev1alpha1.NFSProvisioner{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-nfs",
				Namespace: "test-namespace",
				UID:       "test-uid",
			},
			Spec: cachev1alpha1.NFSProvisionerSpec{
				StorageSize:         "10Gi",
				SCForNFSProvisioner: "nfs",
			},
		}

		// Create fake client
		fakeClient := fake.NewClientBuilder().WithScheme(testScheme).Build()

		// Create StorageClass manager
		baseManager := NewBaseResourceManager(fakeClient, logr.Discard(), testScheme)
		storageClassManager = NewStorageClassManager(baseManager)
	})

	Describe("EnsureResource", func() {
		It("should create StorageClass with correct provisioner", func(ctx SpecContext) {
			err := storageClassManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify StorageClass was created
			sc := &storagev1.StorageClass{}
			err = storageClassManager.Client.Get(ctx, types.NamespacedName{
				Name: "nfs",
			}, sc)
			Expect(err).NotTo(HaveOccurred())

			// Verify StorageClass metadata
			Expect(sc.Name).To(Equal("nfs"))
			Expect(sc.Provisioner).To(Equal("example.com/nfs"))

			// Verify no extra parameters
			Expect(sc.Parameters).To(BeNil())
		})

		It("should use custom storage class name if specified", func(ctx SpecContext) {
			nfsProvisioner.Spec.SCForNFSProvisioner = "custom-nfs-sc"

			err := storageClassManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify StorageClass was created with custom name
			sc := &storagev1.StorageClass{}
			err = storageClassManager.Client.Get(ctx, types.NamespacedName{
				Name: "custom-nfs-sc",
			}, sc)
			Expect(err).NotTo(HaveOccurred())
			Expect(sc.Name).To(Equal("custom-nfs-sc"))
			Expect(sc.Provisioner).To(Equal("example.com/nfs"))
		})

		It("should use default storage class name if not specified", func(ctx SpecContext) {
			nfsProvisioner.Spec.SCForNFSProvisioner = ""

			err := storageClassManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify StorageClass was created with default name
			sc := &storagev1.StorageClass{}
			err = storageClassManager.Client.Get(ctx, types.NamespacedName{
				Name: defaults.SCForNFSProvisioner,
			}, sc)
			Expect(err).NotTo(HaveOccurred())
			Expect(sc.Name).To(Equal(defaults.SCForNFSProvisioner))
		})

		It("should not recreate existing StorageClass", func(ctx SpecContext) {
			// First call
			err := storageClassManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Second call - should be idempotent
			err = storageClassManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify only one StorageClass exists
			sc := &storagev1.StorageClass{}
			err = storageClassManager.Client.Get(ctx, types.NamespacedName{
				Name: "nfs",
			}, sc)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create StorageClass with all required fields", func(ctx SpecContext) {
			err := storageClassManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify StorageClass has all required fields
			sc := &storagev1.StorageClass{}
			err = storageClassManager.Client.Get(ctx, types.NamespacedName{
				Name: "nfs",
			}, sc)
			Expect(err).NotTo(HaveOccurred())

			// Verify basic fields
			Expect(sc.Name).To(Equal("nfs"))
			Expect(sc.Provisioner).To(Equal("example.com/nfs"))
			Expect(sc.Parameters).To(BeNil())
		})
	})

	Describe("GetResourceName", func() {
		It("should return correct resource name", func(ctx SpecContext) {
			Expect(storageClassManager.GetResourceName()).To(Equal("StorageClass"))
		})
	})
})
