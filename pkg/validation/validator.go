package validation

import (
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

// ValidationError represents a validation failure with detailed context.
type ValidationError struct {
	Field   string // The field that failed validation
	Message string // User-friendly error message
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return e.Message
}
