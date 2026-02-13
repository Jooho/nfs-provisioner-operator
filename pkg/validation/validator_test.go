package validation_test

import (
	"testing"
	"time"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/pkg/validation"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestValidation(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Validation Suite")
}

var _ = Describe("Validator", func() {
	var (
		validator validation.Validator
	)

	BeforeEach(func() {
		validator = validation.NewValidator()
	})

	// T017: Test valid configurations (single storage option set)
	Describe("Valid configurations", func() {
		It("should accept hostPathDir as the only storage option", func(ctx SpecContext) {
			nfs := &cachev1alpha1.NFSProvisioner{
				Spec: cachev1alpha1.NFSProvisionerSpec{
					HostPathDir: "/mnt/nfs",
				},
			}
			err := validator.Validate(nfs)
			Expect(err).ToNot(HaveOccurred())
		}, SpecTimeout(10*time.Second))

		It("should accept pvc as the only storage option", func(ctx SpecContext) {
			nfs := &cachev1alpha1.NFSProvisioner{
				Spec: cachev1alpha1.NFSProvisionerSpec{
					Pvc: "nfs-pvc",
				},
			}
			err := validator.Validate(nfs)
			Expect(err).ToNot(HaveOccurred())
		}, SpecTimeout(10*time.Second))

		It("should accept scForNFSPvc as the only storage option", func(ctx SpecContext) {
			nfs := &cachev1alpha1.NFSProvisioner{
				Spec: cachev1alpha1.NFSProvisionerSpec{
					SCForNFSPvc: "fast-storage",
				},
			}
			err := validator.Validate(nfs)
			Expect(err).ToNot(HaveOccurred())
		}, SpecTimeout(10*time.Second))
	})

	// T018: Test invalid configurations (multiple/zero storage options)
	Describe("Invalid configurations", func() {
		It("should reject when no storage option is set", func(ctx SpecContext) {
			nfs := &cachev1alpha1.NFSProvisioner{
				Spec: cachev1alpha1.NFSProvisionerSpec{
					// No storage options set
				},
			}
			err := validator.Validate(nfs)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("exactly one of spec.hostPathDir, spec.pvc, or spec.scForNFSPvc must be set"))
			Expect(err.Error()).To(ContainSubstring("none are set"))
		}, SpecTimeout(10*time.Second))

		It("should reject when both hostPathDir and pvc are set", func(ctx SpecContext) {
			nfs := &cachev1alpha1.NFSProvisioner{
				Spec: cachev1alpha1.NFSProvisionerSpec{
					HostPathDir: "/mnt/nfs",
					Pvc:         "nfs-pvc",
				},
			}
			err := validator.Validate(nfs)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("exactly one of spec.hostPathDir, spec.pvc, or spec.scForNFSPvc must be set"))
			Expect(err.Error()).To(ContainSubstring("multiple options are currently set"))
		}, SpecTimeout(10*time.Second))

		It("should reject when both hostPathDir and scForNFSPvc are set", func(ctx SpecContext) {
			nfs := &cachev1alpha1.NFSProvisioner{
				Spec: cachev1alpha1.NFSProvisionerSpec{
					HostPathDir: "/mnt/nfs",
					SCForNFSPvc: "fast-storage",
				},
			}
			err := validator.Validate(nfs)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("multiple options are currently set"))
		}, SpecTimeout(10*time.Second))

		It("should reject when both pvc and scForNFSPvc are set", func(ctx SpecContext) {
			nfs := &cachev1alpha1.NFSProvisioner{
				Spec: cachev1alpha1.NFSProvisionerSpec{
					Pvc:         "nfs-pvc",
					SCForNFSPvc: "fast-storage",
				},
			}
			err := validator.Validate(nfs)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("multiple options are currently set"))
		}, SpecTimeout(10*time.Second))

		It("should reject when all three storage options are set", func(ctx SpecContext) {
			nfs := &cachev1alpha1.NFSProvisioner{
				Spec: cachev1alpha1.NFSProvisionerSpec{
					HostPathDir: "/mnt/nfs",
					Pvc:         "nfs-pvc",
					SCForNFSPvc: "fast-storage",
				},
			}
			err := validator.Validate(nfs)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("multiple options are currently set"))
		}, SpecTimeout(10*time.Second))
	})
})
