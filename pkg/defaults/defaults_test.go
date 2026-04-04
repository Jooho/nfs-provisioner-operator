package defaults_test

import (
	"testing"
	"time"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/pkg/defaults"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestDefaults(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Defaults Suite")
}

var _ = Describe("ApplyDefaults", func() {
	Describe("StorageSize defaults", func() {
		It("should apply default storageSize when not set", func(ctx SpecContext) {
			nfs := &cachev1alpha1.NFSProvisioner{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-nfs",
					Namespace: "default",
				},
				Spec: cachev1alpha1.NFSProvisionerSpec{},
			}

			defaults.ApplyDefaults(nfs)

			Expect(nfs.Spec.StorageSize).To(Equal("10Gi"))
		}, SpecTimeout(10*time.Second))

		It("should not override existing storageSize", func(ctx SpecContext) {
			nfs := &cachev1alpha1.NFSProvisioner{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-nfs",
					Namespace: "default",
				},
				Spec: cachev1alpha1.NFSProvisionerSpec{
					StorageSize: "20Gi",
				},
			}

			defaults.ApplyDefaults(nfs)

			Expect(nfs.Spec.StorageSize).To(Equal("20Gi"))
		}, SpecTimeout(10*time.Second))
	})

	Describe("SCForNFS defaults", func() {
		It("should apply default scForNFS when not set", func(ctx SpecContext) {
			nfs := &cachev1alpha1.NFSProvisioner{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-nfs",
					Namespace: "default",
				},
				Spec: cachev1alpha1.NFSProvisionerSpec{},
			}

			defaults.ApplyDefaults(nfs)

			Expect(nfs.Spec.SCForNFSProvisioner).To(Equal("nfs"))
		}, SpecTimeout(10*time.Second))

		It("should not override existing scForNFS", func(ctx SpecContext) {
			nfs := &cachev1alpha1.NFSProvisioner{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-nfs",
					Namespace: "default",
				},
				Spec: cachev1alpha1.NFSProvisionerSpec{
					SCForNFSProvisioner: "custom-nfs",
				},
			}

			defaults.ApplyDefaults(nfs)

			Expect(nfs.Spec.SCForNFSProvisioner).To(Equal("custom-nfs"))
		}, SpecTimeout(10*time.Second))
	})

	Describe("NFSImageConfiguration defaults", func() {
		It("should apply default image when NFSImageConfiguration is nil", func(ctx SpecContext) {
			nfs := &cachev1alpha1.NFSProvisioner{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-nfs",
					Namespace: "default",
				},
				Spec: cachev1alpha1.NFSProvisionerSpec{},
			}

			defaults.ApplyDefaults(nfs)

			Expect(nfs.Spec.NFSImageConfiguration).ToNot(BeNil())
			Expect(nfs.Spec.NFSImageConfiguration.Image).ToNot(BeNil())
			Expect(*nfs.Spec.NFSImageConfiguration.Image).To(ContainSubstring("nfs-provisioner"))
		}, SpecTimeout(10*time.Second))

		It("should apply default imagePullPolicy when not set", func(ctx SpecContext) {
			nfs := &cachev1alpha1.NFSProvisioner{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-nfs",
					Namespace: "default",
				},
				Spec: cachev1alpha1.NFSProvisionerSpec{},
			}

			defaults.ApplyDefaults(nfs)

			Expect(nfs.Spec.NFSImageConfiguration.ImagePullPolicy).ToNot(BeNil())
			Expect(*nfs.Spec.NFSImageConfiguration.ImagePullPolicy).To(Equal(corev1.PullIfNotPresent))
		}, SpecTimeout(10*time.Second))

		It("should not override existing image", func(ctx SpecContext) {
			nfs := &cachev1alpha1.NFSProvisioner{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-nfs",
					Namespace: "default",
				},
				Spec: cachev1alpha1.NFSProvisionerSpec{
					NFSImageConfiguration: &cachev1alpha1.ImageConfiguration{
						Image: ptr.To("custom-image:v1"),
					},
				},
			}

			defaults.ApplyDefaults(nfs)

			Expect(*nfs.Spec.NFSImageConfiguration.Image).To(Equal("custom-image:v1"))
		}, SpecTimeout(10*time.Second))

		It("should not override existing imagePullPolicy", func(ctx SpecContext) {
			policy := corev1.PullAlways
			nfs := &cachev1alpha1.NFSProvisioner{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-nfs",
					Namespace: "default",
				},
				Spec: cachev1alpha1.NFSProvisionerSpec{
					NFSImageConfiguration: &cachev1alpha1.ImageConfiguration{
						ImagePullPolicy: &policy,
					},
				},
			}

			defaults.ApplyDefaults(nfs)

			Expect(*nfs.Spec.NFSImageConfiguration.ImagePullPolicy).To(Equal(corev1.PullAlways))
		}, SpecTimeout(10*time.Second))
	})

	Describe("All defaults together", func() {
		It("should apply all defaults to an empty spec", func(ctx SpecContext) {
			nfs := &cachev1alpha1.NFSProvisioner{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-nfs",
					Namespace: "default",
				},
				Spec: cachev1alpha1.NFSProvisionerSpec{},
			}

			defaults.ApplyDefaults(nfs)

			Expect(nfs.Spec.StorageSize).To(Equal("10Gi"))
			Expect(nfs.Spec.SCForNFSProvisioner).To(Equal("nfs"))
			Expect(nfs.Spec.NFSImageConfiguration).ToNot(BeNil())
			Expect(nfs.Spec.NFSImageConfiguration.Image).ToNot(BeNil())
			Expect(nfs.Spec.NFSImageConfiguration.ImagePullPolicy).ToNot(BeNil())
			Expect(*nfs.Spec.NFSImageConfiguration.ImagePullPolicy).To(Equal(corev1.PullIfNotPresent))
		}, SpecTimeout(10*time.Second))
	})
})
