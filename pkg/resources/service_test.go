package resources

import (
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/pkg/defaults"
)

var _ = Describe("ServiceManager", func() {
	var (
		serviceManager *ServiceManager
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
				StorageSize: "10Gi",
			},
		}

		// Create fake client
		fakeClient := fake.NewClientBuilder().WithScheme(testScheme).Build()

		// Create service manager
		baseManager := NewBaseResourceManager(fakeClient, logr.Discard(), testScheme)
		serviceManager = NewServiceManager(baseManager)
	})

	Describe("EnsureResource", func() {
		It("should create service with correct ports", func(ctx SpecContext) {
			err := serviceManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify service was created
			service := &corev1.Service{}
			err = serviceManager.Client.Get(ctx, types.NamespacedName{
				Name:      defaults.Service,
				Namespace: nfsProvisioner.Namespace,
			}, service)
			Expect(err).NotTo(HaveOccurred())

			// Verify service metadata
			Expect(service.Name).To(Equal(defaults.Service))
			Expect(service.Namespace).To(Equal(nfsProvisioner.Namespace))

			// Verify service has required NFS ports
			ports := service.Spec.Ports
			Expect(ports).To(HaveLen(12))

			// Check for essential NFS ports
			var hasNFSPort, hasMountdPort, hasRPCPort bool
			for _, port := range ports {
				if port.Name == "nfs" && port.Port == 2049 {
					hasNFSPort = true
				}
				if port.Name == "mountd" && port.Port == 20048 {
					hasMountdPort = true
				}
				if port.Name == "rpcbind" && port.Port == 111 {
					hasRPCPort = true
				}
			}
			Expect(hasNFSPort).To(BeTrue(), "should have NFS port 2049")
			Expect(hasMountdPort).To(BeTrue(), "should have mountd port 20048")
			Expect(hasRPCPort).To(BeTrue(), "should have rpcbind port 111")

			// Verify selector
			Expect(service.Spec.Selector).To(HaveKeyWithValue("app", defaults.Deployment))
		})

		It("should not recreate existing service", func(ctx SpecContext) {
			// First call
			err := serviceManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Second call - should be idempotent
			err = serviceManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify only one service exists
			service := &corev1.Service{}
			err = serviceManager.Client.Get(ctx, types.NamespacedName{
				Name:      defaults.Service,
				Namespace: nfsProvisioner.Namespace,
			}, service)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create service with owner reference", func(ctx SpecContext) {
			err := serviceManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify service was created with owner reference
			service := &corev1.Service{}
			err = serviceManager.Client.Get(ctx, types.NamespacedName{
				Name:      defaults.Service,
				Namespace: nfsProvisioner.Namespace,
			}, service)
			Expect(err).NotTo(HaveOccurred())

			// Verify owner reference
			Expect(service.OwnerReferences).To(HaveLen(1))
			Expect(service.OwnerReferences[0].Name).To(Equal(nfsProvisioner.Name))
			Expect(service.OwnerReferences[0].UID).To(Equal(nfsProvisioner.UID))
		})
	})

	Describe("GetResourceName", func() {
		It("should return correct resource name", func(ctx SpecContext) {
			Expect(serviceManager.GetResourceName()).To(Equal("Service"))
		})
	})
})
