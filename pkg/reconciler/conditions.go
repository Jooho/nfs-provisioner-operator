package reconciler

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
)

// Condition types for NFSProvisioner
const (
	// ConditionTypeReady indicates that the NFSProvisioner is ready to serve requests.
	// True when all resources are created and the NFS server is running.
	ConditionTypeReady = "Ready"

	// ConditionTypeProgressing indicates that the NFSProvisioner is being reconciled.
	// True during active reconciliation, False when idle.
	ConditionTypeProgressing = "Progressing"

	// ConditionTypeDegraded indicates that the NFSProvisioner is experiencing issues.
	// True when transient errors occur (e.g., API unavailable, conflicts).
	ConditionTypeDegraded = "Degraded"

	// ConditionTypeAvailable indicates that the Deployment has available replicas.
	// True when the NFS server Deployment has at least one available replica.
	ConditionTypeAvailable = "Available"
)

// Condition reasons for NFSProvisioner
const (
	// Ready condition reasons
	ReasonReconciliationSucceeded = "ReconciliationSucceeded"
	ReasonReconciliationFailed    = "ReconciliationFailed"

	// Progressing condition reasons
	ReasonReconciling   = "Reconciling"
	ReasonReconcileIdle = "ReconcileIdle"

	// Degraded condition reasons
	ReasonTransientError  = "TransientError"
	ReasonValidationError = "ValidationError"
	ReasonResourceError   = "ResourceError"

	// Available condition reasons
	ReasonDeploymentAvailable   = "DeploymentAvailable"
	ReasonDeploymentUnavailable = "DeploymentUnavailable"
	ReasonDeploymentNotFound    = "DeploymentNotFound"
)

// SetReadyCondition sets the Ready condition to True when reconciliation succeeds.
//
// This condition indicates that all required resources have been created successfully
// and the NFSProvisioner is ready to serve requests.
//
// Parameters:
//   - nfs: The NFSProvisioner resource to update
//   - generation: The observed generation of the CR
//
// The condition is set with:
//   - Type: Ready
//   - Status: True
//   - Reason: ReconciliationSucceeded
//   - Message: "All resources created successfully"
func SetReadyCondition(nfs *cachev1alpha1.NFSProvisioner, generation int64) {
	apimeta.SetStatusCondition(&nfs.Status.Conditions, metav1.Condition{
		Type:               ConditionTypeReady,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: generation,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonReconciliationSucceeded,
		Message:            "All resources created successfully",
	})
}

// SetReadyConditionFalse sets the Ready condition to False when reconciliation fails.
//
// Parameters:
//   - nfs: The NFSProvisioner resource to update
//   - generation: The observed generation of the CR
//   - reason: The reason for failure (e.g., ReasonValidationError, ReasonResourceError)
//   - message: A human-readable message describing the failure
func SetReadyConditionFalse(nfs *cachev1alpha1.NFSProvisioner, generation int64, reason, message string) {
	apimeta.SetStatusCondition(&nfs.Status.Conditions, metav1.Condition{
		Type:               ConditionTypeReady,
		Status:             metav1.ConditionFalse,
		ObservedGeneration: generation,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	})
}

// SetProgressingCondition sets the Progressing condition to True during reconciliation.
//
// This condition indicates that the controller is actively reconciling the NFSProvisioner.
// It should be set at the start of each reconciliation loop.
//
// Parameters:
//   - nfs: The NFSProvisioner resource to update
//   - generation: The observed generation of the CR
//
// The condition is set with:
//   - Type: Progressing
//   - Status: True
//   - Reason: Reconciling
//   - Message: "Reconciling NFSProvisioner resources"
func SetProgressingCondition(nfs *cachev1alpha1.NFSProvisioner, generation int64) {
	apimeta.SetStatusCondition(&nfs.Status.Conditions, metav1.Condition{
		Type:               ConditionTypeProgressing,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: generation,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonReconciling,
		Message:            "Reconciling NFSProvisioner resources",
	})
}

// SetProgressingConditionFalse sets the Progressing condition to False when reconciliation is complete.
//
// Parameters:
//   - nfs: The NFSProvisioner resource to update
//   - generation: The observed generation of the CR
func SetProgressingConditionFalse(nfs *cachev1alpha1.NFSProvisioner, generation int64) {
	apimeta.SetStatusCondition(&nfs.Status.Conditions, metav1.Condition{
		Type:               ConditionTypeProgressing,
		Status:             metav1.ConditionFalse,
		ObservedGeneration: generation,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonReconcileIdle,
		Message:            "Reconciliation complete",
	})
}

// SetDegradedCondition sets the Degraded condition to True on transient errors.
//
// This condition indicates that the NFSProvisioner is experiencing temporary issues
// that may resolve themselves (e.g., API server unavailable, resource conflicts).
//
// Parameters:
//   - nfs: The NFSProvisioner resource to update
//   - generation: The observed generation of the CR
//   - err: The error that caused degradation
//
// The condition is set with:
//   - Type: Degraded
//   - Status: True
//   - Reason: TransientError
//   - Message: Error message from err
func SetDegradedCondition(nfs *cachev1alpha1.NFSProvisioner, generation int64, err error) {
	apimeta.SetStatusCondition(&nfs.Status.Conditions, metav1.Condition{
		Type:               ConditionTypeDegraded,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: generation,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonTransientError,
		Message:            fmt.Sprintf("Transient error during reconciliation: %v", err),
	})
}

// SetDegradedConditionFalse sets the Degraded condition to False when errors are resolved.
//
// Parameters:
//   - nfs: The NFSProvisioner resource to update
//   - generation: The observed generation of the CR
func SetDegradedConditionFalse(nfs *cachev1alpha1.NFSProvisioner, generation int64) {
	apimeta.SetStatusCondition(&nfs.Status.Conditions, metav1.Condition{
		Type:               ConditionTypeDegraded,
		Status:             metav1.ConditionFalse,
		ObservedGeneration: generation,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonReconciliationSucceeded,
		Message:            "No errors detected",
	})
}

// SetDegradedConditionForValidation sets the Degraded condition for validation errors.
//
// Parameters:
//   - nfs: The NFSProvisioner resource to update
//   - generation: The observed generation of the CR
//   - err: The validation error
func SetDegradedConditionForValidation(nfs *cachev1alpha1.NFSProvisioner, generation int64, err error) {
	apimeta.SetStatusCondition(&nfs.Status.Conditions, metav1.Condition{
		Type:               ConditionTypeDegraded,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: generation,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonValidationError,
		Message:            fmt.Sprintf("Validation failed: %v", err),
	})
}

// SetDegradedConditionForResource sets the Degraded condition for resource errors.
//
// Parameters:
//   - nfs: The NFSProvisioner resource to update
//   - generation: The observed generation of the CR
//   - resourceName: The name of the resource that failed
//   - err: The error that occurred
func SetDegradedConditionForResource(nfs *cachev1alpha1.NFSProvisioner, generation int64, resourceName string, err error) {
	apimeta.SetStatusCondition(&nfs.Status.Conditions, metav1.Condition{
		Type:               ConditionTypeDegraded,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: generation,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonResourceError,
		Message:            fmt.Sprintf("Failed to create/update %s: %v", resourceName, err),
	})
}

// SetAvailableCondition sets the Available condition based on Deployment status.
//
// This condition indicates whether the NFS server Deployment has at least one
// available replica. It queries the Deployment status and sets the condition accordingly.
//
// Parameters:
//   - ctx: The context for the operation
//   - client: Kubernetes client for querying Deployment status
//   - nfs: The NFSProvisioner resource to update
//   - generation: The observed generation of the CR
//
// The condition is set to:
//   - True if Deployment has available replicas
//   - False if Deployment exists but has no available replicas
//   - False if Deployment is not found
//
// Returns:
//   - error: Non-nil if querying Deployment fails
func SetAvailableCondition(ctx context.Context, k8sClient client.Client, nfs *cachev1alpha1.NFSProvisioner, generation int64) error {
	// Query the Deployment to check replica status
	deployment := &appsv1.Deployment{}
	deploymentName := nfs.Name + "-nfs-provisioner"
	err := k8sClient.Get(ctx, client.ObjectKey{
		Namespace: nfs.Namespace,
		Name:      deploymentName,
	}, deployment)
	if err != nil {
		// Deployment not found or error querying - set Available=False
		apimeta.SetStatusCondition(&nfs.Status.Conditions, metav1.Condition{
			Type:               ConditionTypeAvailable,
			Status:             metav1.ConditionFalse,
			ObservedGeneration: generation,
			LastTransitionTime: metav1.Now(),
			Reason:             ReasonDeploymentNotFound,
			Message:            fmt.Sprintf("Deployment %s not found: %v", deploymentName, err),
		})
		return err
	}

	// Check if Deployment has available replicas
	if deployment.Status.AvailableReplicas > 0 {
		apimeta.SetStatusCondition(&nfs.Status.Conditions, metav1.Condition{
			Type:               ConditionTypeAvailable,
			Status:             metav1.ConditionTrue,
			ObservedGeneration: generation,
			LastTransitionTime: metav1.Now(),
			Reason:             ReasonDeploymentAvailable,
			Message:            fmt.Sprintf("Deployment %s has %d available replicas", deploymentName, deployment.Status.AvailableReplicas),
		})
	} else {
		apimeta.SetStatusCondition(&nfs.Status.Conditions, metav1.Condition{
			Type:               ConditionTypeAvailable,
			Status:             metav1.ConditionFalse,
			ObservedGeneration: generation,
			LastTransitionTime: metav1.Now(),
			Reason:             ReasonDeploymentUnavailable,
			Message:            fmt.Sprintf("Deployment %s has no available replicas (desired: %d, ready: %d)", deploymentName, deployment.Status.Replicas, deployment.Status.ReadyReplicas),
		})
	}

	return nil
}
