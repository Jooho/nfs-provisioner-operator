package resources

import (
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/pkg/defaults"
)

var _ = Describe("PVCManager", func() {
	var (
		pvcManager     *PVCManager
		nfsProvisioner *cachev1alpha1.NFSProvisioner
		testScheme     *runtime.Scheme
	)

	BeforeEach(func(ctx SpecContext) {
		// Create scheme
		testScheme = runtime.NewScheme()
		Expect(corev1.AddToScheme(testScheme)).To(Succeed())
		Expect(cachev1alpha1.AddToScheme(testScheme)).To(Succeed())

		// Create test NFSProvisioner instance
		nfsProvisioner = &cachev1alpha1.NFSProvisioner{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-nfs",
				Namespace: "test-namespace",
				UID:       "test-uid",
			},
			Spec: cachev1alpha1.NFSProvisionerSpec{
				StorageSize: "20Gi",
				SCForNFSPvc: "local-storage",
				HostPathDir: "", // Empty to use PVC
			},
		}

		// Create fake client
		fakeClient := fake.NewClientBuilder().WithScheme(testScheme).Build()

		// Create PVC manager
		baseManager := NewBaseResourceManager(fakeClient, logr.Discard(), testScheme)
		pvcManager = NewPVCManager(baseManager)
	})

	Describe("EnsureResource", func() {
		Context("when using PVC storage", func() {
			It("should create PVC with correct storage class", func(ctx SpecContext) {
				err := pvcManager.EnsureResource(ctx, nfsProvisioner)
				Expect(err).NotTo(HaveOccurred())

				// Verify PVC was created
				pvc := &corev1.PersistentVolumeClaim{}
				err = pvcManager.Client.Get(ctx, types.NamespacedName{
					Name:      defaults.Pvc,
					Namespace: nfsProvisioner.Namespace,
				}, pvc)
				Expect(err).NotTo(HaveOccurred())

				// Verify PVC metadata
				Expect(pvc.Name).To(Equal(defaults.Pvc))
				Expect(pvc.Namespace).To(Equal(nfsProvisioner.Namespace))

				// Verify storage class name
				Expect(pvc.Spec.StorageClassName).NotTo(BeNil())
				Expect(*pvc.Spec.StorageClassName).To(Equal("local-storage"))

				// Verify access modes
				Expect(pvc.Spec.AccessModes).To(ContainElement(corev1.ReadWriteOnce))

				// Verify storage size
				expectedSize := resource.MustParse("20Gi")
				actualSize := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
				Expect(actualSize.Equal(expectedSize)).To(BeTrue())
			})

			It("should create PVC with owner reference", func(ctx SpecContext) {
				err := pvcManager.EnsureResource(ctx, nfsProvisioner)
				Expect(err).NotTo(HaveOccurred())

				// Verify PVC was created with owner reference
				pvc := &corev1.PersistentVolumeClaim{}
				err = pvcManager.Client.Get(ctx, types.NamespacedName{
					Name:      defaults.Pvc,
					Namespace: nfsProvisioner.Namespace,
				}, pvc)
				Expect(err).NotTo(HaveOccurred())

				// Verify owner reference
				Expect(pvc.OwnerReferences).To(HaveLen(1))
				Expect(pvc.OwnerReferences[0].Name).To(Equal(nfsProvisioner.Name))
				Expect(pvc.OwnerReferences[0].UID).To(Equal(nfsProvisioner.UID))
			})

			It("should not recreate existing PVC", func(ctx SpecContext) {
				// First call
				err := pvcManager.EnsureResource(ctx, nfsProvisioner)
				Expect(err).NotTo(HaveOccurred())

				// Second call - should be idempotent
				err = pvcManager.EnsureResource(ctx, nfsProvisioner)
				Expect(err).NotTo(HaveOccurred())

				// Verify only one PVC exists
				pvc := &corev1.PersistentVolumeClaim{}
				err = pvcManager.Client.Get(ctx, types.NamespacedName{
					Name:      defaults.Pvc,
					Namespace: nfsProvisioner.Namespace,
				}, pvc)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when using hostPath storage", func() {
			It("should skip PVC creation", func(ctx SpecContext) {
				nfsProvisioner.Spec.HostPathDir = "/mnt/nfs"
				nfsProvisioner.Spec.SCForNFSPvc = ""

				err := pvcManager.EnsureResource(ctx, nfsProvisioner)
				Expect(err).NotTo(HaveOccurred())

				// Verify PVC was NOT created
				pvc := &corev1.PersistentVolumeClaim{}
				err = pvcManager.Client.Get(ctx, types.NamespacedName{
					Name:      defaults.Pvc,
					Namespace: nfsProvisioner.Namespace,
				}, pvc)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when using existing PVC", func() {
			It("should skip PVC creation", func(ctx SpecContext) {
				nfsProvisioner.Spec.Pvc = "existing-pvc"
				nfsProvisioner.Spec.SCForNFSPvc = ""

				err := pvcManager.EnsureResource(ctx, nfsProvisioner)
				Expect(err).NotTo(HaveOccurred())

				// Verify new PVC was NOT created
				pvc := &corev1.PersistentVolumeClaim{}
				err = pvcManager.Client.Get(ctx, types.NamespacedName{
					Name:      defaults.Pvc,
					Namespace: nfsProvisioner.Namespace,
				}, pvc)
				Expect(err).To(HaveOccurred())
			})

			It("should use existing PVC name", func(ctx SpecContext) {
				// Create existing PVC first
				existingPVC := &corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-existing-pvc",
						Namespace: nfsProvisioner.Namespace,
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("10Gi"),
							},
						},
					},
				}
				Expect(pvcManager.Client.Create(ctx, existingPVC)).To(Succeed())

				nfsProvisioner.Spec.Pvc = "my-existing-pvc"
				nfsProvisioner.Spec.SCForNFSPvc = ""

				err := pvcManager.EnsureResource(ctx, nfsProvisioner)
				Expect(err).NotTo(HaveOccurred())

				// Verify existing PVC is still there
				pvc := &corev1.PersistentVolumeClaim{}
				err = pvcManager.Client.Get(ctx, types.NamespacedName{
					Name:      "my-existing-pvc",
					Namespace: nfsProvisioner.Namespace,
				}, pvc)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("GetResourceName", func() {
		It("should return correct resource name", func(ctx SpecContext) {
			Expect(pvcManager.GetResourceName()).To(Equal("PersistentVolumeClaim"))
		})
	})
})
