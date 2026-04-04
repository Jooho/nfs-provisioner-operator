package integration

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/pkg/defaults"
	"github.com/jooho/nfs-provisioner-operator/pkg/reconciler"
)

const (
	timeout  = time.Second * 30
	interval = time.Millisecond * 500
)

var _ = Describe("NFSProvisioner Reconciliation", Serial, Ordered, func() {
	var testNamespace string

	BeforeEach(func(ctx SpecContext) {
		// Clean up any leftover cluster-scoped resources from prior tests
		cleanupClusterScopedResources(ctx)

		// Create a unique namespace for each test to avoid conflicts
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-nfs-",
			},
		}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())
		testNamespace = ns.Name
	})

	AfterEach(func(ctx SpecContext) {
		// Delete all NFSProvisioner CRs in the test namespace first
		nfsList := &cachev1alpha1.NFSProvisionerList{}
		_ = k8sClient.List(ctx, nfsList)
		for i := range nfsList.Items {
			_ = k8sClient.Delete(ctx, &nfsList.Items[i])
		}

		// Wait for CRs to be deleted (finalizers need to run)
		Eventually(func() int {
			list := &cachev1alpha1.NFSProvisionerList{}
			_ = k8sClient.List(ctx, list)
			return len(list.Items)
		}, "15s", "500ms").Should(Equal(0))

		// Clean up cluster-scoped resources
		cleanupClusterScopedResources(ctx)

		// Delete namespace
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
		}
		_ = k8sClient.Delete(ctx, ns)
	})

	Context("when creating a valid NFSProvisioner CR with hostPathDir", func() {
		It("should create all owned resources", func(ctx SpecContext) {
			nfs := &cachev1alpha1.NFSProvisioner{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-nfs",
					Namespace: testNamespace,
				},
				Spec: cachev1alpha1.NFSProvisionerSpec{
					HostPathDir: "/mnt/nfs",
				},
			}

			Expect(k8sClient.Create(ctx, nfs)).To(Succeed())

			nfsKey := types.NamespacedName{Name: nfs.Name, Namespace: testNamespace}

			// Verify ServiceAccount is created
			Eventually(func(g Gomega) {
				sa := &corev1.ServiceAccount{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name: defaults.ServiceAccount, Namespace: testNamespace,
				}, sa)).To(Succeed())
			}).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			// Verify Deployment is created
			Eventually(func(g Gomega) {
				dep := &appsv1.Deployment{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name: defaults.Deployment, Namespace: testNamespace,
				}, dep)).To(Succeed())
				g.Expect(dep.Spec.Template.Spec.Volumes).NotTo(BeEmpty())
			}).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			// Verify Service is created
			Eventually(func(g Gomega) {
				svc := &corev1.Service{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name: defaults.Service, Namespace: testNamespace,
				}, svc)).To(Succeed())
			}).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			// Verify ClusterRole is created
			Eventually(func(g Gomega) {
				cr := &rbacv1.ClusterRole{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name: defaults.ClusterRole,
				}, cr)).To(Succeed())
			}).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			// Verify ClusterRoleBinding is created
			Eventually(func(g Gomega) {
				crb := &rbacv1.ClusterRoleBinding{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name: defaults.ClusterRoleBinding,
				}, crb)).To(Succeed())
			}).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			// Verify Role is created
			Eventually(func(g Gomega) {
				role := &rbacv1.Role{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name: defaults.Role, Namespace: testNamespace,
				}, role)).To(Succeed())
			}).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			// Verify RoleBinding is created
			Eventually(func(g Gomega) {
				rb := &rbacv1.RoleBinding{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name: defaults.RoleBinding, Namespace: testNamespace,
				}, rb)).To(Succeed())
			}).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			// Verify StorageClass is created
			Eventually(func(g Gomega) {
				sc := &storagev1.StorageClass{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name: defaults.SCForNFSProvisioner,
				}, sc)).To(Succeed())
			}).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			// Verify status conditions are set
			Eventually(func(g Gomega) {
				updatedNFS := &cachev1alpha1.NFSProvisioner{}
				g.Expect(k8sClient.Get(ctx, nfsKey, updatedNFS)).To(Succeed())
				g.Expect(updatedNFS.Status.Conditions).NotTo(BeEmpty())

				readyCond := apimeta.FindStatusCondition(updatedNFS.Status.Conditions, reconciler.ConditionTypeReady)
				g.Expect(readyCond).NotTo(BeNil())
				g.Expect(readyCond.Status).To(Equal(metav1.ConditionTrue))
				g.Expect(readyCond.Reason).To(Equal(reconciler.ReasonReconciliationSucceeded))

				g.Expect(updatedNFS.Status.Phase).To(Equal(reconciler.PhaseReady))
				// Note: ObservedGeneration is only set when Deployment is available,
				// which doesn't happen in envtest (no real pod scheduling).
			}).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(Succeed())
		}, SpecTimeout(60*time.Second))
	})

	Context("when updating an NFSProvisioner CR", func() {
		It("should reconcile the updated resources", func(ctx SpecContext) {
			nfs := &cachev1alpha1.NFSProvisioner{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-nfs-update",
					Namespace: testNamespace,
				},
				Spec: cachev1alpha1.NFSProvisionerSpec{
					HostPathDir: "/mnt/nfs",
				},
			}

			Expect(k8sClient.Create(ctx, nfs)).To(Succeed())

			nfsKey := types.NamespacedName{Name: nfs.Name, Namespace: testNamespace}

			// Wait for initial reconciliation to complete
			Eventually(func(g Gomega) {
				updatedNFS := &cachev1alpha1.NFSProvisioner{}
				g.Expect(k8sClient.Get(ctx, nfsKey, updatedNFS)).To(Succeed())
				readyCond := apimeta.FindStatusCondition(updatedNFS.Status.Conditions, reconciler.ConditionTypeReady)
				g.Expect(readyCond).NotTo(BeNil())
				g.Expect(readyCond.Status).To(Equal(metav1.ConditionTrue))
			}).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			// Update the CR's storageSize with retry to handle conflicts
			Eventually(func(g Gomega) {
				latest := &cachev1alpha1.NFSProvisioner{}
				g.Expect(k8sClient.Get(ctx, nfsKey, latest)).To(Succeed())
				latest.Spec.StorageSize = "20Gi"
				g.Expect(k8sClient.Update(ctx, latest)).To(Succeed())
			}).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			// Verify reconciliation runs again with updated generation
			Eventually(func(g Gomega) {
				latestNFS := &cachev1alpha1.NFSProvisioner{}
				g.Expect(k8sClient.Get(ctx, nfsKey, latestNFS)).To(Succeed())
				readyCond := apimeta.FindStatusCondition(latestNFS.Status.Conditions, reconciler.ConditionTypeReady)
				g.Expect(readyCond).NotTo(BeNil())
				g.Expect(readyCond.Status).To(Equal(metav1.ConditionTrue))
			}).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(Succeed())
		}, SpecTimeout(60*time.Second))
	})

	Context("when deleting an NFSProvisioner CR", func() {
		It("should run finalizer and clean up cluster-scoped resources", func(ctx SpecContext) {
			nfs := &cachev1alpha1.NFSProvisioner{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-nfs-delete",
					Namespace: testNamespace,
				},
				Spec: cachev1alpha1.NFSProvisionerSpec{
					HostPathDir: "/mnt/nfs",
				},
			}

			Expect(k8sClient.Create(ctx, nfs)).To(Succeed())

			nfsKey := types.NamespacedName{Name: nfs.Name, Namespace: testNamespace}

			// Wait for reconciliation to complete
			Eventually(func(g Gomega) {
				updatedNFS := &cachev1alpha1.NFSProvisioner{}
				g.Expect(k8sClient.Get(ctx, nfsKey, updatedNFS)).To(Succeed())
				readyCond := apimeta.FindStatusCondition(updatedNFS.Status.Conditions, reconciler.ConditionTypeReady)
				g.Expect(readyCond).NotTo(BeNil())
				g.Expect(readyCond.Status).To(Equal(metav1.ConditionTrue))
			}).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			// Delete the NFSProvisioner CR
			toDelete := &cachev1alpha1.NFSProvisioner{}
			Expect(k8sClient.Get(ctx, nfsKey, toDelete)).To(Succeed())
			Expect(k8sClient.Delete(ctx, toDelete)).To(Succeed())

			// Verify the CR is deleted (finalizer should run and remove it)
			Eventually(func(g Gomega) {
				deletedNFS := &cachev1alpha1.NFSProvisioner{}
				err := k8sClient.Get(ctx, nfsKey, deletedNFS)
				g.Expect(err).To(HaveOccurred())
			}).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			// Verify cluster-scoped resources are cleaned up by the finalizer
			Eventually(func(g Gomega) {
				cr := &rbacv1.ClusterRole{}
				err := k8sClient.Get(ctx, types.NamespacedName{Name: defaults.ClusterRole}, cr)
				g.Expect(err).To(HaveOccurred())
			}).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			Eventually(func(g Gomega) {
				crb := &rbacv1.ClusterRoleBinding{}
				err := k8sClient.Get(ctx, types.NamespacedName{Name: defaults.ClusterRoleBinding}, crb)
				g.Expect(err).To(HaveOccurred())
			}).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(Succeed())
		}, SpecTimeout(60*time.Second))
	})

	Context("when creating an invalid NFSProvisioner CR", func() {
		It("should set error status condition for multiple storage options", func(ctx SpecContext) {
			nfs := &cachev1alpha1.NFSProvisioner{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-nfs-invalid",
					Namespace: testNamespace,
				},
				Spec: cachev1alpha1.NFSProvisionerSpec{
					HostPathDir: "/mnt/nfs",
					Pvc:         "my-pvc",
				},
			}

			Expect(k8sClient.Create(ctx, nfs)).To(Succeed())

			nfsKey := types.NamespacedName{Name: nfs.Name, Namespace: testNamespace}

			Eventually(func(g Gomega) {
				updatedNFS := &cachev1alpha1.NFSProvisioner{}
				g.Expect(k8sClient.Get(ctx, nfsKey, updatedNFS)).To(Succeed())
				g.Expect(updatedNFS.Status.Conditions).NotTo(BeEmpty())

				readyCond := apimeta.FindStatusCondition(updatedNFS.Status.Conditions, reconciler.ConditionTypeReady)
				g.Expect(readyCond).NotTo(BeNil())
				g.Expect(readyCond.Status).To(Equal(metav1.ConditionFalse))

				degradedCond := apimeta.FindStatusCondition(updatedNFS.Status.Conditions, reconciler.ConditionTypeDegraded)
				g.Expect(degradedCond).NotTo(BeNil())
				g.Expect(degradedCond.Status).To(Equal(metav1.ConditionTrue))
				g.Expect(degradedCond.Reason).To(Equal(reconciler.ReasonValidationError))

				g.Expect(updatedNFS.Status.Phase).To(Equal(reconciler.PhaseFailed))
			}).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(Succeed())
		}, SpecTimeout(60*time.Second))

		It("should set error status condition for invalid storageSize format", func(ctx SpecContext) {
			nfs := &cachev1alpha1.NFSProvisioner{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-nfs-bad-size",
					Namespace: testNamespace,
				},
				Spec: cachev1alpha1.NFSProvisionerSpec{
					HostPathDir: "/mnt/nfs",
					StorageSize: "not-a-valid-size",
				},
			}

			Expect(k8sClient.Create(ctx, nfs)).To(Succeed())

			nfsKey := types.NamespacedName{Name: nfs.Name, Namespace: testNamespace}

			Eventually(func(g Gomega) {
				updatedNFS := &cachev1alpha1.NFSProvisioner{}
				g.Expect(k8sClient.Get(ctx, nfsKey, updatedNFS)).To(Succeed())
				g.Expect(updatedNFS.Status.Conditions).NotTo(BeEmpty())

				readyCond := apimeta.FindStatusCondition(updatedNFS.Status.Conditions, reconciler.ConditionTypeReady)
				g.Expect(readyCond).NotTo(BeNil())
				g.Expect(readyCond.Status).To(Equal(metav1.ConditionFalse))

				g.Expect(updatedNFS.Status.Phase).To(Equal(reconciler.PhaseFailed))
			}).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(Succeed())
		}, SpecTimeout(60*time.Second))
	})

	Context("when creating NFSProvisioner with scForNFSPvc", func() {
		It("should create PVC for NFS server", func(ctx SpecContext) {
			nfs := &cachev1alpha1.NFSProvisioner{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-nfs-sc",
					Namespace: testNamespace,
				},
				Spec: cachev1alpha1.NFSProvisionerSpec{
					SCForNFSPvc: "standard",
					StorageSize: "5Gi",
				},
			}

			Expect(k8sClient.Create(ctx, nfs)).To(Succeed())

			Eventually(func(g Gomega) {
				pvc := &corev1.PersistentVolumeClaim{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name: defaults.Pvc, Namespace: testNamespace,
				}, pvc)).To(Succeed())

				g.Expect(pvc.Spec.StorageClassName).NotTo(BeNil())
				g.Expect(*pvc.Spec.StorageClassName).To(Equal("standard"))
			}).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(Succeed())
		}, SpecTimeout(60*time.Second))
	})

	Context("idempotency", func() {
		It("should handle multiple reconciliations without error", func(ctx SpecContext) {
			nfs := &cachev1alpha1.NFSProvisioner{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-nfs-idempotent",
					Namespace: testNamespace,
				},
				Spec: cachev1alpha1.NFSProvisionerSpec{
					HostPathDir: "/mnt/nfs",
				},
			}

			Expect(k8sClient.Create(ctx, nfs)).To(Succeed())

			nfsKey := types.NamespacedName{Name: nfs.Name, Namespace: testNamespace}

			// Wait for initial reconciliation to succeed
			Eventually(func(g Gomega) {
				updatedNFS := &cachev1alpha1.NFSProvisioner{}
				g.Expect(k8sClient.Get(ctx, nfsKey, updatedNFS)).To(Succeed())
				readyCond := apimeta.FindStatusCondition(updatedNFS.Status.Conditions, reconciler.ConditionTypeReady)
				g.Expect(readyCond).NotTo(BeNil())
				g.Expect(readyCond.Status).To(Equal(metav1.ConditionTrue))
			}).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			// Trigger another reconciliation by adding a label (with retry for conflicts)
			Eventually(func(g Gomega) {
				latest := &cachev1alpha1.NFSProvisioner{}
				g.Expect(k8sClient.Get(ctx, nfsKey, latest)).To(Succeed())
				if latest.Labels == nil {
					latest.Labels = make(map[string]string)
				}
				latest.Labels["test"] = "idempotent"
				g.Expect(k8sClient.Update(ctx, latest)).To(Succeed())
			}).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			// Verify reconciliation succeeds again with no errors
			Eventually(func(g Gomega) {
				latestNFS := &cachev1alpha1.NFSProvisioner{}
				g.Expect(k8sClient.Get(ctx, nfsKey, latestNFS)).To(Succeed())
				readyCond := apimeta.FindStatusCondition(latestNFS.Status.Conditions, reconciler.ConditionTypeReady)
				g.Expect(readyCond).NotTo(BeNil())
				g.Expect(readyCond.Status).To(Equal(metav1.ConditionTrue))
			}).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			// Verify resources still exist
			dep := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: defaults.Deployment, Namespace: testNamespace,
			}, dep)).To(Succeed())

			svc := &corev1.Service{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: defaults.Service, Namespace: testNamespace,
			}, svc)).To(Succeed())
		}, SpecTimeout(60*time.Second))
	})
})

// cleanupClusterScopedResources removes cluster-scoped resources created during tests.
func cleanupClusterScopedResources(ctx context.Context) {
	cr := &rbacv1.ClusterRole{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: defaults.ClusterRole}, cr); err == nil {
		_ = k8sClient.Delete(ctx, cr)
	}

	crb := &rbacv1.ClusterRoleBinding{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: defaults.ClusterRoleBinding}, crb); err == nil {
		_ = k8sClient.Delete(ctx, crb)
	}

	sc := &storagev1.StorageClass{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: defaults.SCForNFSProvisioner}, sc); err == nil {
		_ = k8sClient.Delete(ctx, sc)
	}

	// Brief wait for resources to be deleted
	time.Sleep(500 * time.Millisecond)
}
