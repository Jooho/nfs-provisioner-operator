package resources

import (
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/pkg/defaults"
)

var _ = Describe("DeploymentManager", func() {
	var (
		deploymentManager *DeploymentManager
		nfsProvisioner    *cachev1alpha1.NFSProvisioner
		testScheme        *runtime.Scheme
	)

	BeforeEach(func(ctx SpecContext) {
		// Create scheme
		testScheme = runtime.NewScheme()
		Expect(corev1.AddToScheme(testScheme)).To(Succeed())
		Expect(appsv1.AddToScheme(testScheme)).To(Succeed())
		Expect(cachev1alpha1.AddToScheme(testScheme)).To(Succeed())

		// Create test NFSProvisioner instance
		nfsProvisioner = &cachev1alpha1.NFSProvisioner{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-nfs",
				Namespace: "test-namespace",
				UID:       "test-uid",
			},
			Spec: cachev1alpha1.NFSProvisionerSpec{
				HostPathDir: "/mnt/nfs",
				StorageSize: "10Gi",
				NFSImageConfiguration: &cachev1alpha1.ImageConfiguration{
					Image:           ptr.To("quay.io/nfs-provisioner:latest"),
					ImagePullPolicy: ptr.To(corev1.PullIfNotPresent),
				},
			},
		}

		// Create fake client
		fakeClient := fake.NewClientBuilder().WithScheme(testScheme).Build()

		// Create deployment manager
		baseManager := NewBaseResourceManager(fakeClient, logr.Discard(), testScheme)
		deploymentManager = NewDeploymentManager(baseManager)
	})

	Describe("EnsureResource", func() {
		It("should create deployment with hostPath volume", func(ctx SpecContext) {
			err := deploymentManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify deployment was created
			deployment := &appsv1.Deployment{}
			err = deploymentManager.Client.Get(ctx, types.NamespacedName{
				Name:      defaults.Deployment,
				Namespace: nfsProvisioner.Namespace,
			}, deployment)
			Expect(err).NotTo(HaveOccurred())

			// Verify deployment metadata
			Expect(deployment.Name).To(Equal(defaults.Deployment))
			Expect(deployment.Namespace).To(Equal(nfsProvisioner.Namespace))

			// Verify volume configuration (hostPath)
			volumes := deployment.Spec.Template.Spec.Volumes
			Expect(volumes).To(HaveLen(1))
			Expect(volumes[0].Name).To(Equal("export-volume"))
			Expect(volumes[0].HostPath).NotTo(BeNil())
			Expect(volumes[0].HostPath.Path).To(Equal("/mnt/nfs"))

			// Verify container configuration
			containers := deployment.Spec.Template.Spec.Containers
			Expect(containers).To(HaveLen(1))
			Expect(containers[0].Image).To(Equal("quay.io/nfs-provisioner:latest"))
			Expect(containers[0].ImagePullPolicy).To(Equal(corev1.PullIfNotPresent))

			// Verify volume mount
			Expect(containers[0].VolumeMounts).To(HaveLen(1))
			Expect(containers[0].VolumeMounts[0].Name).To(Equal("export-volume"))
			Expect(containers[0].VolumeMounts[0].MountPath).To(Equal("/export"))
		})

		It("should create deployment with PVC volume when using PVC storage", func(ctx SpecContext) {
			nfsProvisioner.Spec.HostPathDir = ""
			nfsProvisioner.Spec.SCForNFSPvc = "local-storage"

			err := deploymentManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify deployment was created
			deployment := &appsv1.Deployment{}
			err = deploymentManager.Client.Get(ctx, types.NamespacedName{
				Name:      defaults.Deployment,
				Namespace: nfsProvisioner.Namespace,
			}, deployment)
			Expect(err).NotTo(HaveOccurred())

			// Verify volume configuration (PVC)
			volumes := deployment.Spec.Template.Spec.Volumes
			Expect(volumes).To(HaveLen(1))
			Expect(volumes[0].Name).To(Equal("export-volume"))
			Expect(volumes[0].PersistentVolumeClaim).NotTo(BeNil())
			Expect(volumes[0].PersistentVolumeClaim.ClaimName).To(Equal(defaults.Pvc))
		})

		It("should create deployment with existing PVC when specified", func(ctx SpecContext) {
			nfsProvisioner.Spec.HostPathDir = ""
			nfsProvisioner.Spec.Pvc = "existing-pvc"

			err := deploymentManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify deployment was created
			deployment := &appsv1.Deployment{}
			err = deploymentManager.Client.Get(ctx, types.NamespacedName{
				Name:      defaults.Deployment,
				Namespace: nfsProvisioner.Namespace,
			}, deployment)
			Expect(err).NotTo(HaveOccurred())

			// Verify volume configuration uses existing PVC
			volumes := deployment.Spec.Template.Spec.Volumes
			Expect(volumes).To(HaveLen(1))
			Expect(volumes[0].PersistentVolumeClaim).NotTo(BeNil())
			Expect(volumes[0].PersistentVolumeClaim.ClaimName).To(Equal("existing-pvc"))
		})

		It("should not recreate existing deployment", func(ctx SpecContext) {
			// First call
			err := deploymentManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Second call - should be idempotent
			err = deploymentManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify only one deployment exists
			deployment := &appsv1.Deployment{}
			err = deploymentManager.Client.Get(ctx, types.NamespacedName{
				Name:      defaults.Deployment,
				Namespace: nfsProvisioner.Namespace,
			}, deployment)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should apply default image configuration if not specified", func(ctx SpecContext) {
			nfsProvisioner.Spec.NFSImageConfiguration = nil

			err := deploymentManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify deployment was created with default image
			deployment := &appsv1.Deployment{}
			err = deploymentManager.Client.Get(ctx, types.NamespacedName{
				Name:      defaults.Deployment,
				Namespace: nfsProvisioner.Namespace,
			}, deployment)
			Expect(err).NotTo(HaveOccurred())

			// Verify default image is used
			containers := deployment.Spec.Template.Spec.Containers
			Expect(containers).To(HaveLen(1))
			Expect(containers[0].Image).NotTo(BeEmpty())
		})
	})

	Describe("GetResourceName", func() {
		It("should return correct resource name", func(ctx SpecContext) {
			Expect(deploymentManager.GetResourceName()).To(Equal("Deployment"))
		})
	})
})
