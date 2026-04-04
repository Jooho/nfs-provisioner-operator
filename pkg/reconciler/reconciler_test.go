package reconciler

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/pkg/resources"
	"github.com/jooho/nfs-provisioner-operator/pkg/validation"
)

func TestReconciler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Reconciler Suite")
}

var _ = Describe("Reconciler", func() {
	var (
		testScheme     *runtime.Scheme
		nfsProvisioner *cachev1alpha1.NFSProvisioner
	)

	BeforeEach(func(ctx SpecContext) {
		// Create scheme
		testScheme = runtime.NewScheme()
		Expect(corev1.AddToScheme(testScheme)).To(Succeed())
		Expect(appsv1.AddToScheme(testScheme)).To(Succeed())
		Expect(rbacv1.AddToScheme(testScheme)).To(Succeed())
		Expect(storagev1.AddToScheme(testScheme)).To(Succeed())
		Expect(cachev1alpha1.AddToScheme(testScheme)).To(Succeed())
		Expect(apiextensionsv1.AddToScheme(testScheme)).To(Succeed())

		// Create test NFSProvisioner instance
		nfsProvisioner = &cachev1alpha1.NFSProvisioner{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "test-nfs",
				Namespace:  "test-namespace",
				Generation: 1,
			},
			Spec: cachev1alpha1.NFSProvisionerSpec{
				HostPathDir: "/mnt/nfs",
				StorageSize: "10Gi",
				NFSImageConfiguration: &cachev1alpha1.ImageConfiguration{
					Image:           ptr.To("test-image:v1"),
					ImagePullPolicy: ptr.To(corev1.PullIfNotPresent),
				},
			},
		}
	})

	Describe("Successful Reconciliation", func() {
		It("should reconcile valid CR and create all resources", func(ctx SpecContext) {
			// Create fake client
			fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(nfsProvisioner).WithStatusSubresource(nfsProvisioner).Build()

			// Create validator
			validator := validation.NewValidator()

			// Create resource managers
			resourceManagerSet := resources.NewResourceManagerSet(fakeClient, logr.Discard(), testScheme)
			managers := []resources.ResourceManager{
				resourceManagerSet.ServiceAccount,
				resourceManagerSet.RBAC,
				resourceManagerSet.PVC,
				resourceManagerSet.Service,
				resourceManagerSet.Deployment,
				resourceManagerSet.StorageClass,
			}

			// Create reconciler
			reconciler := NewReconciler(fakeClient, validator, managers, logr.Discard())

			// Reconcile
			result, err := reconciler.Reconcile(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
			Expect(result.RequeueAfter).To(BeZero())

			// Verify status was updated
			updatedNFS := &cachev1alpha1.NFSProvisioner{}
			err = fakeClient.Get(ctx, client.ObjectKeyFromObject(nfsProvisioner), updatedNFS)
			Expect(err).NotTo(HaveOccurred())

			// Verify status fields
			Expect(updatedNFS.Status.Phase).To(Equal(PhaseReady))
			Expect(updatedNFS.Status.ObservedGeneration).To(Equal(int64(1)))

			// Verify Ready condition
			readyCondition := findCondition(updatedNFS.Status.Conditions, ConditionTypeReady)
			Expect(readyCondition).NotTo(BeNil())
			Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue))
			Expect(readyCondition.Reason).To(Equal(ReasonReconciliationSucceeded))

			// Verify Progressing condition is false
			progressingCondition := findCondition(updatedNFS.Status.Conditions, ConditionTypeProgressing)
			Expect(progressingCondition).NotTo(BeNil())
			Expect(progressingCondition.Status).To(Equal(metav1.ConditionFalse))

			// Verify Degraded condition is false
			degradedCondition := findCondition(updatedNFS.Status.Conditions, ConditionTypeDegraded)
			Expect(degradedCondition).NotTo(BeNil())
			Expect(degradedCondition.Status).To(Equal(metav1.ConditionFalse))
		})
	})

	Describe("Validation Failure", func() {
		It("should not requeue when validation fails", func(ctx SpecContext) {
			// Create invalid NFSProvisioner (multiple storage options set - validation error)
			invalidNFS := nfsProvisioner.DeepCopy()
			invalidNFS.Spec.HostPathDir = "/mnt/nfs"
			invalidNFS.Spec.SCForNFSPvc = "local-storage" // Both hostPath and scForNFSPvc set - invalid

			// Create fake client
			fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(invalidNFS).WithStatusSubresource(invalidNFS).Build()

			// Create validator
			validator := validation.NewValidator()

			// Create reconciler with empty managers
			reconciler := NewReconciler(fakeClient, validator, []resources.ResourceManager{}, logr.Discard())

			// Reconcile
			result, err := reconciler.Reconcile(ctx, invalidNFS)
			Expect(err).NotTo(HaveOccurred()) // No error returned for validation failures
			Expect(result.Requeue).To(BeFalse())
			Expect(result.RequeueAfter).To(BeZero())

			// Verify status was updated
			updatedNFS := &cachev1alpha1.NFSProvisioner{}
			err = fakeClient.Get(ctx, client.ObjectKeyFromObject(invalidNFS), updatedNFS)
			Expect(err).NotTo(HaveOccurred())

			// Verify status reflects validation error
			Expect(updatedNFS.Status.Phase).To(Equal(PhaseFailed))
			Expect(updatedNFS.Status.ObservedGeneration).To(Equal(int64(1)))

			// Verify Ready condition is false
			readyCondition := findCondition(updatedNFS.Status.Conditions, ConditionTypeReady)
			Expect(readyCondition).NotTo(BeNil())
			Expect(readyCondition.Status).To(Equal(metav1.ConditionFalse))
			Expect(readyCondition.Reason).To(Equal(ReasonValidationError))

			// Verify Degraded condition is true
			degradedCondition := findCondition(updatedNFS.Status.Conditions, ConditionTypeDegraded)
			Expect(degradedCondition).NotTo(BeNil())
			Expect(degradedCondition.Status).To(Equal(metav1.ConditionTrue))
			Expect(degradedCondition.Reason).To(Equal(ReasonValidationError))
		})
	})

	Describe("Resource Creation Failure", func() {
		It("should requeue with error on transient resource creation failure", func(ctx SpecContext) {
			// Create fake client that will fail on create
			fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(nfsProvisioner).WithStatusSubresource(nfsProvisioner).Build()

			// Create validator
			validator := validation.NewValidator()

			// Create failing resource manager
			failingManager := &failingResourceManager{
				resourceName: "TestResource",
				errorType:    errorTypeTransient,
			}
			managers := []resources.ResourceManager{failingManager}

			// Create reconciler
			reconciler := NewReconciler(fakeClient, validator, managers, logr.Discard())

			// Reconcile
			result, err := reconciler.Reconcile(ctx, nfsProvisioner)
			Expect(err).To(HaveOccurred()) // Error returned for transient failures
			Expect(result.Requeue).To(BeFalse())
			Expect(result.RequeueAfter).To(BeZero())

			// Verify status was updated
			updatedNFS := &cachev1alpha1.NFSProvisioner{}
			err = fakeClient.Get(ctx, client.ObjectKeyFromObject(nfsProvisioner), updatedNFS)
			Expect(err).NotTo(HaveOccurred())

			// Verify status reflects transient error
			Expect(updatedNFS.Status.Phase).To(Equal(PhaseProgressing))

			// Verify Ready condition is false
			readyCondition := findCondition(updatedNFS.Status.Conditions, ConditionTypeReady)
			Expect(readyCondition).NotTo(BeNil())
			Expect(readyCondition.Status).To(Equal(metav1.ConditionFalse))
			Expect(readyCondition.Reason).To(Equal(ReasonResourceError))

			// Verify Degraded condition is true
			degradedCondition := findCondition(updatedNFS.Status.Conditions, ConditionTypeDegraded)
			Expect(degradedCondition).NotTo(BeNil())
			Expect(degradedCondition.Status).To(Equal(metav1.ConditionTrue))
		})

		It("should requeue after delay on permanent resource creation failure", func(ctx SpecContext) {
			// Create fake client
			fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(nfsProvisioner).WithStatusSubresource(nfsProvisioner).Build()

			// Create validator
			validator := validation.NewValidator()

			// Create failing resource manager (permanent error)
			failingManager := &failingResourceManager{
				resourceName: "TestResource",
				errorType:    errorTypePermanent,
			}
			managers := []resources.ResourceManager{failingManager}

			// Create reconciler
			reconciler := NewReconciler(fakeClient, validator, managers, logr.Discard())

			// Reconcile
			result, err := reconciler.Reconcile(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred()) // No error for permanent failures
			Expect(result.Requeue).To(BeFalse())
			Expect(result.RequeueAfter).NotTo(BeZero()) // Should have delay

			// Verify status was updated
			updatedNFS := &cachev1alpha1.NFSProvisioner{}
			err = fakeClient.Get(ctx, client.ObjectKeyFromObject(nfsProvisioner), updatedNFS)
			Expect(err).NotTo(HaveOccurred())

			// Verify status reflects permanent error
			Expect(updatedNFS.Status.Phase).To(Equal(PhaseFailed))

			// Verify Ready condition is false
			readyCondition := findCondition(updatedNFS.Status.Conditions, ConditionTypeReady)
			Expect(readyCondition).NotTo(BeNil())
			Expect(readyCondition.Status).To(Equal(metav1.ConditionFalse))
			Expect(readyCondition.Reason).To(Equal(ReasonResourceError))

			// Verify Degraded condition is true
			degradedCondition := findCondition(updatedNFS.Status.Conditions, ConditionTypeDegraded)
			Expect(degradedCondition).NotTo(BeNil())
			Expect(degradedCondition.Status).To(Equal(metav1.ConditionTrue))
			Expect(degradedCondition.Reason).To(Equal(ReasonResourceError))
		})
	})

	Describe("CR Update Scenario", func() {
		It("should handle CR updates correctly", func(ctx SpecContext) {
			// Create fake client
			fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(nfsProvisioner).WithStatusSubresource(nfsProvisioner).Build()

			// Create validator
			validator := validation.NewValidator()

			// Create resource managers
			resourceManagerSet := resources.NewResourceManagerSet(fakeClient, logr.Discard(), testScheme)
			managers := []resources.ResourceManager{
				resourceManagerSet.ServiceAccount,
			}

			// Create reconciler
			reconciler := NewReconciler(fakeClient, validator, managers, logr.Discard())

			// First reconcile
			result, err := reconciler.Reconcile(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			// Simulate CR update (change generation)
			nfsProvisioner.Generation = 2
			nfsProvisioner.Spec.StorageSize = "20Gi"
			err = fakeClient.Update(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Second reconcile
			result, err = reconciler.Reconcile(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			// Verify status was updated with new generation
			updatedNFS := &cachev1alpha1.NFSProvisioner{}
			err = fakeClient.Get(ctx, client.ObjectKeyFromObject(nfsProvisioner), updatedNFS)
			Expect(err).NotTo(HaveOccurred())

			Expect(updatedNFS.Status.ObservedGeneration).To(Equal(int64(2)))
			Expect(updatedNFS.Status.Phase).To(Equal(PhaseReady))
		})
	})

	Describe("Status Condition Updates", func() {
		It("should update all status conditions correctly", func(ctx SpecContext) {
			// Create fake client
			fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(nfsProvisioner).WithStatusSubresource(nfsProvisioner).Build()

			// Create validator
			validator := validation.NewValidator()

			// Create minimal managers
			resourceManagerSet := resources.NewResourceManagerSet(fakeClient, logr.Discard(), testScheme)
			managers := []resources.ResourceManager{
				resourceManagerSet.ServiceAccount,
			}

			// Create reconciler
			reconciler := NewReconciler(fakeClient, validator, managers, logr.Discard())

			// Reconcile
			result, err := reconciler.Reconcile(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			// Verify status was updated
			updatedNFS := &cachev1alpha1.NFSProvisioner{}
			err = fakeClient.Get(ctx, client.ObjectKeyFromObject(nfsProvisioner), updatedNFS)
			Expect(err).NotTo(HaveOccurred())

			// Verify all condition types are present
			conditions := updatedNFS.Status.Conditions
			Expect(len(conditions)).To(BeNumerically(">=", 3))

			// Verify Ready condition
			readyCondition := findCondition(conditions, ConditionTypeReady)
			Expect(readyCondition).NotTo(BeNil())
			Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue))
			Expect(readyCondition.ObservedGeneration).To(Equal(int64(1)))

			// Verify Progressing condition
			progressingCondition := findCondition(conditions, ConditionTypeProgressing)
			Expect(progressingCondition).NotTo(BeNil())
			Expect(progressingCondition.Status).To(Equal(metav1.ConditionFalse))
			Expect(progressingCondition.ObservedGeneration).To(Equal(int64(1)))

			// Verify Degraded condition
			degradedCondition := findCondition(conditions, ConditionTypeDegraded)
			Expect(degradedCondition).NotTo(BeNil())
			Expect(degradedCondition.Status).To(Equal(metav1.ConditionFalse))
			Expect(degradedCondition.ObservedGeneration).To(Equal(int64(1)))
		})
	})

	Describe("Error Classification", func() {
		It("should classify conflict as transient", func(ctx SpecContext) {
			err := apierrors.NewConflict(schema.GroupResource{}, "test", errors.New("conflict"))
			Expect(classifyError(err)).To(Equal(errorTypeTransient))
		})

		It("should classify forbidden as permanent", func(ctx SpecContext) {
			err := apierrors.NewForbidden(schema.GroupResource{}, "test", errors.New("forbidden"))
			Expect(classifyError(err)).To(Equal(errorTypePermanent))
		})

		It("should classify timeout as transient", func(ctx SpecContext) {
			err := apierrors.NewTimeoutError("test", 1)
			Expect(classifyError(err)).To(Equal(errorTypeTransient))
		})

		It("should classify invalid as permanent", func(ctx SpecContext) {
			err := apierrors.NewInvalid(schema.GroupKind{}, "test", nil)
			Expect(classifyError(err)).To(Equal(errorTypePermanent))
		})

		It("should classify unknown error as transient", func(ctx SpecContext) {
			err := errors.New("unknown error")
			Expect(classifyError(err)).To(Equal(errorTypeTransient))
		})
	})
})

// Helper functions

// findCondition finds a condition by type in the conditions list
func findCondition(conditions []metav1.Condition, conditionType string) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}
	return nil
}

// failingResourceManager is a mock ResourceManager that always fails
type failingResourceManager struct {
	resourceName string
	errorType    errorType
}

func (m *failingResourceManager) GetResourceName() string {
	return m.resourceName
}

func (m *failingResourceManager) EnsureResource(ctx context.Context, nfs *cachev1alpha1.NFSProvisioner) error {
	switch m.errorType {
	case errorTypeTransient:
		return apierrors.NewConflict(schema.GroupResource{}, "test", errors.New("simulated transient error"))
	case errorTypePermanent:
		return apierrors.NewForbidden(schema.GroupResource{}, "test", errors.New("simulated permanent error"))
	default:
		return errors.New("simulated unknown error")
	}
}
