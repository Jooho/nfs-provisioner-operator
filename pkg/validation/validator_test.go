package validation_test

import (
	"testing"
	"time"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/pkg/validation"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/ptr"
)

func TestValidation(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Validation Suite")
}

var _ = Describe("Validator", func() {
	var validator validation.Validator

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
		It("should accept when no storage option is set (uses default StorageClass)", func(ctx SpecContext) {
			nfs := &cachev1alpha1.NFSProvisioner{
				Spec: cachev1alpha1.NFSProvisionerSpec{
					// No storage options set - operator uses cluster default SC
				},
			}
			err := validator.Validate(nfs)
			Expect(err).NotTo(HaveOccurred())
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

	// T016: Test field format validation
	Describe("Field format validation", func() {
		Describe("StorageSize validation", func() {
			It("should accept valid Kubernetes quantities", func(ctx SpecContext) {
				validSizes := []string{"10Gi", "1Ti", "500Mi", "1G", "100M", "1000000"}
				for _, size := range validSizes {
					nfs := &cachev1alpha1.NFSProvisioner{
						Spec: cachev1alpha1.NFSProvisionerSpec{
							HostPathDir: "/mnt/nfs",
							StorageSize: size,
						},
					}
					err := validator.Validate(nfs)
					Expect(err).ToNot(HaveOccurred(), "Expected %s to be valid", size)
				}
			}, SpecTimeout(10*time.Second))

			It("should reject invalid quantities", func(ctx SpecContext) {
				nfs := &cachev1alpha1.NFSProvisioner{
					Spec: cachev1alpha1.NFSProvisionerSpec{
						HostPathDir: "/mnt/nfs",
						StorageSize: "invalid-size",
					},
				}
				err := validator.Validate(nfs)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.storageSize must be a valid Kubernetes quantity"))
				Expect(err.Error()).To(ContainSubstring("10Gi"))
			}, SpecTimeout(10*time.Second))
		})

		Describe("StorageClass name validation", func() {
			It("should accept valid DNS subdomain names", func(ctx SpecContext) {
				validNames := []string{"nfs", "nfs-storage", "storage.class.name", "nfs123"}
				for _, name := range validNames {
					nfs := &cachev1alpha1.NFSProvisioner{
						Spec: cachev1alpha1.NFSProvisionerSpec{
							HostPathDir:         "/mnt/nfs",
							SCForNFSProvisioner: name,
						},
					}
					err := validator.Validate(nfs)
					Expect(err).ToNot(HaveOccurred(), "Expected %s to be valid", name)
				}
			}, SpecTimeout(10*time.Second))

			It("should reject invalid DNS names", func(ctx SpecContext) {
				nfs := &cachev1alpha1.NFSProvisioner{
					Spec: cachev1alpha1.NFSProvisionerSpec{
						HostPathDir:         "/mnt/nfs",
						SCForNFSProvisioner: "Invalid_Name",
					},
				}
				err := validator.Validate(nfs)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.scForNFSProvisioner must be a valid Kubernetes resource name"))
			}, SpecTimeout(10*time.Second))
		})

		Describe("Image validation", func() {
			It("should accept valid image references", func(ctx SpecContext) {
				validImages := []string{
					"nginx",
					"nginx:latest",
					"quay.io/repository/image:v1.0",
					"docker.io/library/nginx:1.21",
					"k8s.gcr.io/sig-storage/nfs-provisioner:v4.0.0",
					"registry.example.com:5000/repo/image:tag",
					"repo/image@sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
				}
				for _, image := range validImages {
					nfs := &cachev1alpha1.NFSProvisioner{
						Spec: cachev1alpha1.NFSProvisionerSpec{
							HostPathDir: "/mnt/nfs",
							NFSImageConfiguration: &cachev1alpha1.ImageConfiguration{
								Image: ptr.To(image),
							},
						},
					}
					err := validator.Validate(nfs)
					Expect(err).ToNot(HaveOccurred(), "Expected %s to be valid", image)
				}
			}, SpecTimeout(10*time.Second))

			It("should reject invalid image references", func(ctx SpecContext) {
				nfs := &cachev1alpha1.NFSProvisioner{
					Spec: cachev1alpha1.NFSProvisionerSpec{
						HostPathDir: "/mnt/nfs",
						NFSImageConfiguration: &cachev1alpha1.ImageConfiguration{
							Image: ptr.To("INVALID IMAGE"),
						},
					},
				}
				err := validator.Validate(nfs)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("spec.nfsImageConfiguration.image must be a valid container image reference"))
			}, SpecTimeout(10*time.Second))

			It("should accept nil image (will use default)", func(ctx SpecContext) {
				nfs := &cachev1alpha1.NFSProvisioner{
					Spec: cachev1alpha1.NFSProvisionerSpec{
						HostPathDir: "/mnt/nfs",
						NFSImageConfiguration: &cachev1alpha1.ImageConfiguration{
							Image: nil,
						},
					},
				}
				err := validator.Validate(nfs)
				Expect(err).ToNot(HaveOccurred())
			}, SpecTimeout(10*time.Second))
		})
	})
})
