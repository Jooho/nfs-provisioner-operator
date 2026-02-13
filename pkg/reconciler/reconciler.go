package reconciler

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/pkg/defaults"
	"github.com/jooho/nfs-provisioner-operator/pkg/resources"
	"github.com/jooho/nfs-provisioner-operator/pkg/validation"
)

// Reconciler orchestrates the reconciliation logic for NFSProvisioner resources.
// It coordinates validation, defaults application, and resource management in a modular way.
type Reconciler interface {
	// Reconcile handles the reconciliation logic for an NFSProvisioner resource.
	//
	// Parameters:
	//   - ctx: The context for the reconciliation request
	//   - nfs: The NFSProvisioner resource to reconcile
	//
	// Returns:
	//   - ctrl.Result: The result of the reconciliation (requeue, delay, etc.)
	//   - error: Non-nil if reconciliation failed
	Reconcile(ctx context.Context, nfs *cachev1alpha1.NFSProvisioner) (ctrl.Result, error)
}

// reconciler is the default implementation of the Reconciler interface.
type reconciler struct {
	client    client.Client
	validator validation.Validator
	managers  []resources.ResourceManager
	logger    logr.Logger
}

// NewReconciler creates a new Reconciler with dependency injection.
//
// Parameters:
//   - client: Kubernetes client for resource operations
//   - validator: Validator for NFSProvisioner resources
//   - managers: List of ResourceManagers for creating/updating resources
//   - logger: Logger for structured logging
//
// Returns:
//   - Reconciler: A new reconciler instance
func NewReconciler(
	client client.Client,
	validator validation.Validator,
	managers []resources.ResourceManager,
	logger logr.Logger,
) Reconciler {
	return &reconciler{
		client:    client,
		validator: validator,
		managers:  managers,
		logger:    logger,
	}
}

// Reconcile implements the Reconciler interface.
// It orchestrates validation → defaults → resource creation → status update.
func (r *reconciler) Reconcile(ctx context.Context, nfs *cachev1alpha1.NFSProvisioner) (ctrl.Result, error) {
	log := r.logger.WithValues("nfsprovisioner", client.ObjectKeyFromObject(nfs))

	// Step 1: Apply defaults to unset fields
	defaults.ApplyDefaults(nfs)
	log.Info("Applied default values to NFSProvisioner spec")

	// Step 2: Validate the resource
	if err := r.validator.Validate(nfs); err != nil {
		log.Error(err, "Validation failed for NFSProvisioner")
		return r.handleValidationError(ctx, nfs, err)
	}
	log.Info("Validation succeeded for NFSProvisioner")

	// Step 3: Ensure all required resources exist
	for _, manager := range r.managers {
		if err := manager.EnsureResource(ctx, nfs); err != nil {
			log.Error(err, "Failed to ensure resource", "resource", manager.GetResourceName())
			return r.handleResourceError(ctx, nfs, manager.GetResourceName(), err)
		}
		log.Info("Ensured resource exists", "resource", manager.GetResourceName())
	}

	// Step 4: Update status to reflect successful reconciliation
	// (Status update logic will be added in Phase 4: User Story 2)

	log.Info("Reconciliation completed successfully")
	return ctrl.Result{}, nil
}

// handleValidationError handles validation errors (permanent errors - no retry).
// Validation errors are user errors that require CR modification.
func (r *reconciler) handleValidationError(ctx context.Context, nfs *cachev1alpha1.NFSProvisioner, err error) (ctrl.Result, error) {
	r.logger.Info("Validation error - no requeue", "error", err.Error())
	// Do not requeue for validation errors - user must fix the CR
	// In Phase 4 (User Story 2), we'll set status.Conditions with Degraded=True
	return ctrl.Result{}, nil
}

// handleResourceError handles resource creation/update errors.
// These may be transient (API server unavailable) or permanent (quota exceeded).
func (r *reconciler) handleResourceError(ctx context.Context, nfs *cachev1alpha1.NFSProvisioner, resourceName string, err error) (ctrl.Result, error) {
	errorType := classifyError(err)

	switch errorType {
	case errorTypeTransient:
		// Transient errors: Let controller-runtime handle exponential backoff
		r.logger.Info("Transient error - will retry with exponential backoff",
			"resource", resourceName,
			"error", err.Error())
		return ctrl.Result{}, err

	case errorTypePermanent:
		// Permanent errors: Requeue after 5 minutes
		r.logger.Info("Permanent error - will retry after 5 minutes",
			"resource", resourceName,
			"error", err.Error())
		// In Phase 4 (User Story 2), we'll set status.Conditions with Degraded=True
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil

	default:
		// Unknown errors: Treat as transient
		r.logger.Info("Unknown error - treating as transient",
			"resource", resourceName,
			"error", err.Error())
		return ctrl.Result{}, err
	}
}

// errorType represents the classification of an error.
type errorType int

const (
	errorTypeUnknown errorType = iota
	errorTypeTransient
	errorTypePermanent
)

// classifyError classifies an error as transient or permanent.
//
// Transient errors:
//   - Connection errors (API server unavailable)
//   - Conflict errors (resource version mismatch)
//   - Timeout errors
//
// Permanent errors:
//   - Forbidden errors (RBAC issue)
//   - Invalid errors (malformed request)
//   - AlreadyExists errors (user-created resource conflict)
func classifyError(err error) errorType {
	if err == nil {
		return errorTypeUnknown
	}

	// Kubernetes API errors
	if errors.IsConflict(err) {
		return errorTypeTransient
	}
	if errors.IsTimeout(err) {
		return errorTypeTransient
	}
	if errors.IsServerTimeout(err) {
		return errorTypeTransient
	}
	if errors.IsServiceUnavailable(err) {
		return errorTypeTransient
	}
	if errors.IsTooManyRequests(err) {
		return errorTypeTransient
	}
	if errors.IsInternalError(err) {
		return errorTypeTransient
	}

	// Permanent errors
	if errors.IsForbidden(err) {
		return errorTypePermanent
	}
	if errors.IsInvalid(err) {
		return errorTypePermanent
	}
	if errors.IsUnauthorized(err) {
		return errorTypePermanent
	}
	if errors.IsMethodNotSupported(err) {
		return errorTypePermanent
	}
	if errors.IsAlreadyExists(err) {
		return errorTypePermanent
	}

	// Check for corev1.Event creation errors (can be ignored)
	if isEventError(err) {
		return errorTypeTransient
	}

	// Unknown errors - treat as transient
	return errorTypeTransient
}

// isEventError checks if the error is related to Event creation.
// Event creation failures are non-critical and should not block reconciliation.
func isEventError(err error) bool {
	// Simple heuristic: check if error message contains "Event"
	// More sophisticated check could use error wrapping
	errMsg := err.Error()
	return len(errMsg) > 0 && (errMsg[0:5] == "Event" ||
		len(errMsg) > 10 && errMsg[len(errMsg)-5:] == "Event")
}

// ReconcilerStatus represents the current state of reconciliation.
// This will be expanded in Phase 4 (User Story 2) to include Conditions.
type ReconcilerStatus struct {
	Phase             string
	ObservedGeneration int64
	Message           string
}

// newReconcilerStatus creates a ReconcilerStatus from an NFSProvisioner resource.
func newReconcilerStatus(nfs *cachev1alpha1.NFSProvisioner, phase string, message string) ReconcilerStatus {
	return ReconcilerStatus{
		Phase:             phase,
		ObservedGeneration: nfs.Generation,
		Message:           message,
	}
}

// String returns a human-readable representation of the ReconcilerStatus.
func (s ReconcilerStatus) String() string {
	return fmt.Sprintf("Phase=%s, ObservedGeneration=%d, Message=%s",
		s.Phase, s.ObservedGeneration, s.Message)
}
