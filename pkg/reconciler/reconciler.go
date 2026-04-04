package reconciler

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/pkg/defaults"
	"github.com/jooho/nfs-provisioner-operator/pkg/resources"
	"github.com/jooho/nfs-provisioner-operator/pkg/validation"
)

// Phase constants for NFSProvisioner lifecycle
const (
	// PhasePending indicates the NFSProvisioner is pending reconciliation
	PhasePending = "Pending"

	// PhaseProgressing indicates the NFSProvisioner is being reconciled
	PhaseProgressing = "Progressing"

	// PhaseReady indicates the NFSProvisioner is ready and all resources are created
	PhaseReady = "Ready"

	// PhaseFailed indicates the NFSProvisioner has encountered a permanent error
	PhaseFailed = "Failed"
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
	logger    logr.Logger
	managers  []resources.ResourceManager
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
	log := r.logger.WithValues(
		"nfsprovisioner", client.ObjectKeyFromObject(nfs),
		"generation", nfs.Generation,
	)

	generation := nfs.Generation

	// Skip reconciliation if already reconciled for this generation.
	// Status updates trigger new reconcile events via resourceVersion change;
	// without this guard the reconciler loops infinitely.
	if nfs.Status.ObservedGeneration == generation && nfs.Status.Phase == PhaseReady {
		log.V(1).Info("Already reconciled for this generation, skipping")
		return ctrl.Result{}, nil
	}

	log.Info("Started reconciliation")

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
	SetReadyCondition(nfs, generation)
	SetProgressingConditionFalse(nfs, generation)
	SetDegradedConditionFalse(nfs, generation)
	nfs.Status.Phase = PhaseReady

	// Step 5: Check Deployment availability and set Available condition
	deploymentAvailable := IsDeploymentAvailable(ctx, r.client, nfs)
	if err := SetAvailableCondition(ctx, r.client, nfs, generation); err != nil {
		log.V(1).Info("Deployment not found yet, will requeue", "error", err.Error())
	}

	// Only set ObservedGeneration when fully complete (including Deployment available).
	// This allows the ObservedGeneration guard to let requeued reconciles through.
	if deploymentAvailable {
		nfs.Status.ObservedGeneration = generation
	}

	// Step 6: Update final status
	if err := r.updateStatus(ctx, nfs); err != nil {
		log.Error(err, "Failed to update final status")
		return ctrl.Result{}, err
	}

	if !deploymentAvailable {
		log.Info("Resources created, waiting for Deployment to become available")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	log.Info("Reconciliation completed successfully", "phase", nfs.Status.Phase)
	return ctrl.Result{}, nil
}

// updateStatus updates the NFSProvisioner status subresource.
//
// This function uses the status writer to update only the status subresource,
// not the entire CR. This is important for proper Kubernetes reconciliation.
//
// Parameters:
//   - ctx: The context for the operation
//   - nfs: The NFSProvisioner resource with updated status fields
//
// Returns:
//   - error: Non-nil if the status update fails
func (r *reconciler) updateStatus(ctx context.Context, nfs *cachev1alpha1.NFSProvisioner) error {
	// Re-fetch latest to avoid conflict with concurrent modifications (e.g., finalizer).
	// On failure, controller-runtime automatically retries the full Reconcile with a fresh object.
	latest := &cachev1alpha1.NFSProvisioner{}
	if err := r.client.Get(ctx, client.ObjectKeyFromObject(nfs), latest); err != nil {
		return err
	}

	latest.Status.Phase = nfs.Status.Phase
	latest.Status.Conditions = nfs.Status.Conditions
	latest.Status.ObservedGeneration = nfs.Status.ObservedGeneration
	latest.Status.Nodes = nfs.Status.Nodes

	if err := r.client.Status().Update(ctx, latest); err != nil {
		r.logger.Error(err, "Failed to update NFSProvisioner status",
			"namespace", nfs.Namespace,
			"name", nfs.Name,
			"phase", nfs.Status.Phase,
		)
		return err
	}
	return nil
}

// handleValidationError handles validation errors (permanent errors - no retry).
// Validation errors are user errors that require CR modification.
func (r *reconciler) handleValidationError(ctx context.Context, nfs *cachev1alpha1.NFSProvisioner, err error) (ctrl.Result, error) {
	generation := nfs.Generation
	r.logger.Info("Validation error - no requeue",
		"error", err.Error(),
		"namespace", nfs.Namespace,
		"name", nfs.Name,
	)

	// Set status conditions to reflect validation failure
	SetReadyConditionFalse(nfs, generation, ReasonValidationError, err.Error())
	SetProgressingConditionFalse(nfs, generation)
	SetDegradedConditionForValidation(nfs, generation, err)
	nfs.Status.Phase = PhaseFailed
	// Do NOT set ObservedGeneration on error - it means "successfully processed"

	// Update status to reflect validation error
	if updateErr := r.updateStatus(ctx, nfs); updateErr != nil {
		r.logger.Error(updateErr, "Failed to update status after validation error")
		// Return update error to trigger retry
		return ctrl.Result{}, updateErr
	}

	// Do not requeue for validation errors - user must fix the CR
	return ctrl.Result{}, nil
}

// handleResourceError handles resource creation/update errors.
// These may be transient (API server unavailable) or permanent (quota exceeded).
func (r *reconciler) handleResourceError(ctx context.Context, nfs *cachev1alpha1.NFSProvisioner, resourceName string, err error) (ctrl.Result, error) {
	generation := nfs.Generation
	errorType := classifyError(err)

	switch errorType {
	case errorTypeTransient:
		// Transient errors: Let controller-runtime handle exponential backoff
		r.logger.Info("Transient error - will retry with exponential backoff",
			"resource", resourceName,
			"error", err.Error(),
			"namespace", nfs.Namespace,
			"name", nfs.Name,
		)

		// Set status conditions to reflect transient error
		SetReadyConditionFalse(nfs, generation, ReasonResourceError, fmt.Sprintf("Failed to create/update %s: %v", resourceName, err))
		SetDegradedCondition(nfs, generation, err)
		nfs.Status.Phase = PhaseProgressing // Still progressing, will retry

		// Update status
		if updateErr := r.updateStatus(ctx, nfs); updateErr != nil {
			r.logger.Error(updateErr, "Failed to update status after transient error")
		}

		// Return error to trigger exponential backoff
		return ctrl.Result{}, err

	case errorTypePermanent:
		// Permanent errors: Requeue after 5 minutes
		r.logger.Info("Permanent error - will retry after 5 minutes",
			"resource", resourceName,
			"error", err.Error(),
			"namespace", nfs.Namespace,
			"name", nfs.Name,
		)

		// Set status conditions to reflect permanent error
		SetReadyConditionFalse(nfs, generation, ReasonResourceError, fmt.Sprintf("Failed to create/update %s: %v", resourceName, err))
		SetProgressingConditionFalse(nfs, generation)
		SetDegradedConditionForResource(nfs, generation, resourceName, err)
		nfs.Status.Phase = PhaseFailed

		// Update status
		if updateErr := r.updateStatus(ctx, nfs); updateErr != nil {
			r.logger.Error(updateErr, "Failed to update status after permanent error")
			// Continue with requeue anyway
		}

		// Requeue after delay for permanent errors
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil

	default:
		// Unknown errors: Treat as transient
		r.logger.Info("Unknown error - treating as transient",
			"resource", resourceName,
			"error", err.Error(),
			"namespace", nfs.Namespace,
			"name", nfs.Name,
		)

		// Set status conditions for unknown error (treat as transient)
		SetReadyConditionFalse(nfs, generation, ReasonResourceError, fmt.Sprintf("Failed to create/update %s: %v", resourceName, err))
		SetDegradedCondition(nfs, generation, err)
		nfs.Status.Phase = PhaseProgressing

		// Update status
		if updateErr := r.updateStatus(ctx, nfs); updateErr != nil {
			r.logger.Error(updateErr, "Failed to update status after unknown error")
		}

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

	// Unknown errors - treat as transient
	return errorTypeTransient
}
