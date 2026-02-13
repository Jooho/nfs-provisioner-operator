package validation

import (
	"fmt"
	"regexp"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/validation"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
)

// Validator validates NFSProvisioner resources according to business rules.
// Validation failures should result in user-friendly error messages that guide
// users to fix the issue.
type Validator interface {
	// Validate checks if the NFSProvisioner resource is valid.
	//
	// Parameters:
	//   - nfs: The NFSProvisioner resource to validate
	//
	// Returns:
	//   - error: Non-nil if validation fails, with a descriptive message
	Validate(nfs *cachev1alpha1.NFSProvisioner) error
}

// validator is the default implementation of the Validator interface.
type validator struct{}

// NewValidator creates a new Validator instance.
func NewValidator() Validator {
	return &validator{}
}

// Validate implements the Validator interface.
// It validates NFSProvisioner resources according to the rules defined in data-model.md.
func (v *validator) Validate(nfs *cachev1alpha1.NFSProvisioner) error {
	if err := v.validateStorageOptions(nfs); err != nil {
		return err
	}
	if err := v.validateStorageSize(nfs); err != nil {
		return err
	}
	if err := v.validateStorageClassName(nfs); err != nil {
		return err
	}
	if err := v.validateImage(nfs); err != nil {
		return err
	}
	return nil
}

// validateStorageOptions ensures exactly one storage option is set.
// Rule: Exactly one of spec.hostPathDir, spec.pvc, or spec.scForNFSPvc must be set.
func (v *validator) validateStorageOptions(nfs *cachev1alpha1.NFSProvisioner) error {
	pvc := nfs.Spec.Pvc
	sc := nfs.Spec.SCForNFSPvc
	hostPathDir := nfs.Spec.HostPathDir

	// Count how many storage options are set
	setCount := 0
	if pvc != "" {
		setCount++
	}
	if sc != "" {
		setCount++
	}
	if hostPathDir != "" {
		setCount++
	}

	if setCount == 0 {
		return &ValidationError{
			Field:   "spec.storage",
			Message: "exactly one of spec.hostPathDir, spec.pvc, or spec.scForNFSPvc must be set; currently none are set",
		}
	}

	if setCount > 1 {
		return &ValidationError{
			Field:   "spec.storage",
			Message: "exactly one of spec.hostPathDir, spec.pvc, or spec.scForNFSPvc must be set; multiple options are currently set",
		}
	}

	return nil
}

// validateStorageSize validates spec.storageSize is a valid Kubernetes quantity.
// Rule: If set, must be a valid quantity like '10Gi', '1Ti', '500Mi'.
func (v *validator) validateStorageSize(nfs *cachev1alpha1.NFSProvisioner) error {
	storageSize := nfs.Spec.StorageSize
	if storageSize == "" {
		// Empty is valid - defaults will be applied
		return nil
	}

	if _, err := resource.ParseQuantity(storageSize); err != nil {
		return &ValidationError{
			Field:   "spec.storageSize",
			Message: fmt.Sprintf("spec.storageSize must be a valid Kubernetes quantity (e.g., '10Gi', '1Ti', '500Mi'); got '%s'", storageSize),
		}
	}

	return nil
}

// validateStorageClassName validates spec.scForNFSProvisioner is a valid Kubernetes resource name.
// Rule: If set, must be a valid DNS subdomain (alphanumeric, '-', '.', max 253 chars).
func (v *validator) validateStorageClassName(nfs *cachev1alpha1.NFSProvisioner) error {
	scName := nfs.Spec.SCForNFSProvisioner
	if scName == "" {
		// Empty is valid - defaults will be applied
		return nil
	}

	if errs := validation.IsDNS1123Subdomain(scName); len(errs) > 0 {
		return &ValidationError{
			Field:   "spec.scForNFSProvisioner",
			Message: fmt.Sprintf("spec.scForNFSProvisioner must be a valid Kubernetes resource name (DNS subdomain: lowercase alphanumeric, '-', '.', max 253 chars); validation errors: %v", errs),
		}
	}

	return nil
}

// validateImage validates spec.nfsImageConfiguration.image is a valid container image reference.
// Rule: Must match standard image reference format (optional registry, repository, optional tag/digest).
func (v *validator) validateImage(nfs *cachev1alpha1.NFSProvisioner) error {
	if nfs.Spec.NFSImageConfiguration == nil || nfs.Spec.NFSImageConfiguration.Image == nil {
		// Nil is valid - defaults will be applied
		return nil
	}

	image := *nfs.Spec.NFSImageConfiguration.Image
	if image == "" {
		// Empty is valid - defaults will be applied
		return nil
	}

	// Basic image format validation: should have at least a repository name
	// Format: [registry/]repository[:tag|@digest]
	imageRegex := regexp.MustCompile(`^([a-z0-9._-]+(\.[a-z0-9._-]+)*(:[0-9]+)?/)?[a-z0-9._-]+(/[a-z0-9._-]+)*(:[a-zA-Z0-9._-]+|@sha256:[a-f0-9]{64})?$`)
	if !imageRegex.MatchString(image) {
		return &ValidationError{
			Field:   "spec.nfsImageConfiguration.image",
			Message: fmt.Sprintf("spec.nfsImageConfiguration.image must be a valid container image reference (e.g., 'quay.io/repo/image:tag' or 'image:latest'); got '%s'", image),
		}
	}

	return nil
}

// ValidationError represents a validation failure with detailed context.
type ValidationError struct {
	Field   string // The field that failed validation
	Message string // User-friendly error message
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return e.Message
}
