package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/pkg/defaults"
	"github.com/jooho/nfs-provisioner-operator/pkg/reconciler"
)

const (
	timeout  = time.Minute * 3
	interval = time.Second * 2

	e2eNamespace = "e2e-nfs-test"
)

// nfsDeploymentReady tracks whether the NFS Deployment reached ready state.
var nfsDeploymentReady bool

// isOCP returns true when E2E_PLATFORM=ocp
func isOCP() bool {
	return strings.EqualFold(os.Getenv("E2E_PLATFORM"), "ocp")
}

// writerImage returns a lightweight image for the writer pod test.
func writerImage() string {
	if isOCP() {
		return "registry.access.redhat.com/ubi9/ubi-minimal:latest"
	}
	return "busybox:1.36"
}

// hostPathForPlatform returns the platform-specific hostPath directory.
func hostPathForPlatform() string {
	if isOCP() {
		return "/home/core/nfs"
	}
	return "/tmp/nfs-e2e"
}

var _ = Describe("NFS Provisioner E2E - hostPathDir mode", Ordered, func() {
	BeforeAll(func(ctx SpecContext) {
		By("creating e2e test namespace")
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: e2eNamespace},
		}
		err := k8sClient.Create(ctx, ns)
		if err != nil && !errors.IsAlreadyExists(err) {
			Expect(err).NotTo(HaveOccurred())
		}

		By("preparing cluster nodes (directory + label)")
		if isOCP() {
			prepareOCPNodes()
		} else {
			prepareKindNodes()
		}
	}, NodeTimeout(120*time.Second))

	AfterAll(func(ctx SpecContext) {
		By("cleaning up e2e resources")

		nfs := &cachev1alpha1.NFSProvisioner{
			ObjectMeta: metav1.ObjectMeta{Name: "e2e-nfs", Namespace: e2eNamespace},
		}
		_ = k8sClient.Delete(ctx, nfs)

		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name: "e2e-nfs", Namespace: e2eNamespace,
			}, &cachev1alpha1.NFSProvisioner{})
			return errors.IsNotFound(err)
		}, "30s", "1s").Should(BeTrue())

		_ = k8sClient.Delete(ctx, &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{Name: defaults.ClusterRole},
		})
		_ = k8sClient.Delete(ctx, &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: defaults.ClusterRoleBinding},
		})
		_ = k8sClient.Delete(ctx, &storagev1.StorageClass{
			ObjectMeta: metav1.ObjectMeta{Name: defaults.SCForNFSProvisioner},
		})
		_ = k8sClient.Delete(ctx, &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: "e2e-test-pvc", Namespace: e2eNamespace},
		})
		_ = k8sClient.Delete(ctx, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "e2e-nfs-writer", Namespace: e2eNamespace},
		})
		_ = k8sClient.Delete(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: e2eNamespace},
		})
	}, NodeTimeout(60*time.Second))

	Context("deploy operator and create CR", func() {
		It("should create NFSProvisioner CR and reconcile all resources", func(ctx SpecContext) {
			hostPathDir := hostPathForPlatform()

			By(fmt.Sprintf("creating NFSProvisioner CR (platform=%s, hostPath=%s)", os.Getenv("E2E_PLATFORM"), hostPathDir))
			nfs := &cachev1alpha1.NFSProvisioner{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "e2e-nfs",
					Namespace: e2eNamespace,
				},
				Spec: cachev1alpha1.NFSProvisionerSpec{
					HostPathDir:  hostPathDir,
					StorageSize:  "1Gi",
					NodeSelector: map[string]string{"app": "nfs-provisioner"},
				},
			}
			Expect(k8sClient.Create(ctx, nfs)).To(Succeed())

			nfsKey := types.NamespacedName{Name: "e2e-nfs", Namespace: e2eNamespace}

			By("verifying status reaches Ready")
			Eventually(func(g Gomega) {
				latest := &cachev1alpha1.NFSProvisioner{}
				g.Expect(k8sClient.Get(ctx, nfsKey, latest)).To(Succeed())
				readyCond := apimeta.FindStatusCondition(latest.Status.Conditions, reconciler.ConditionTypeReady)
				g.Expect(readyCond).NotTo(BeNil())
				g.Expect(readyCond.Status).To(Equal(metav1.ConditionTrue))
				g.Expect(latest.Status.Phase).To(Equal(reconciler.PhaseReady))
			}).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			By("verifying all resources exist")
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: defaults.Deployment, Namespace: e2eNamespace,
			}, &appsv1.Deployment{})).To(Succeed())

			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: defaults.Service, Namespace: e2eNamespace,
			}, &corev1.Service{})).To(Succeed())

			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: defaults.SCForNFSProvisioner,
			}, &storagev1.StorageClass{})).To(Succeed())

			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: defaults.ServiceAccount, Namespace: e2eNamespace,
			}, &corev1.ServiceAccount{})).To(Succeed())

			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: defaults.ClusterRole,
			}, &rbacv1.ClusterRole{})).To(Succeed())

			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: defaults.ClusterRoleBinding,
			}, &rbacv1.ClusterRoleBinding{})).To(Succeed())
		}, SpecTimeout(timeout))
	})

	Context("NFS Deployment readiness", func() {
		It("should have the NFS server pod running", func(ctx SpecContext) {
			Eventually(func(g Gomega) {
				dep := &appsv1.Deployment{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name: defaults.Deployment, Namespace: e2eNamespace,
				}, dep)).To(Succeed())
				g.Expect(dep.Status.AvailableReplicas).To(BeNumerically(">=", 1),
					fmt.Sprintf("available=%d, ready=%d",
						dep.Status.AvailableReplicas, dep.Status.ReadyReplicas))
			}).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			nfsDeploymentReady = true
		}, SpecTimeout(timeout))
	})

	Context("NFS volume provisioning", func() {
		It("should create PV when PVC is created using NFS StorageClass", func(ctx SpecContext) {
			if !nfsDeploymentReady {
				Skip("NFS server Deployment not ready")
			}

			scName := defaults.SCForNFSProvisioner
			pvc := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "e2e-test-pvc",
					Namespace: e2eNamespace,
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					StorageClassName: &scName,
					AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("100Mi"),
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, pvc)).To(Succeed())

			Eventually(func(g Gomega) {
				foundPVC := &corev1.PersistentVolumeClaim{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name: "e2e-test-pvc", Namespace: e2eNamespace,
				}, foundPVC)).To(Succeed())
				g.Expect(foundPVC.Status.Phase).To(Equal(corev1.ClaimBound),
					fmt.Sprintf("PVC phase: %s", foundPVC.Status.Phase))
			}).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(Succeed())
		}, SpecTimeout(timeout))

		It("should allow a pod to mount the NFS PVC and write data", func(ctx SpecContext) {
			if !nfsDeploymentReady {
				Skip("NFS server Deployment not ready")
			}

			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "e2e-nfs-writer",
					Namespace: e2eNamespace,
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:    "writer",
							Image:   writerImage(),
							Command: []string{"sh", "-c", "echo 'e2e-test-data' > /mnt/nfs/testfile.txt && cat /mnt/nfs/testfile.txt && sleep 5"},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "nfs-vol", MountPath: "/mnt/nfs"},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "nfs-vol",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "e2e-test-pvc",
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, pod)).To(Succeed())

			Eventually(func(g Gomega) {
				foundPod := &corev1.Pod{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name: "e2e-nfs-writer", Namespace: e2eNamespace,
				}, foundPod)).To(Succeed())
				g.Expect(foundPod.Status.Phase).To(Equal(corev1.PodSucceeded),
					fmt.Sprintf("Pod phase: %s", foundPod.Status.Phase))
			}).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(Succeed())
		}, SpecTimeout(timeout))
	})

	Context("CR deletion cleanup", func() {
		It("should clean up cluster-scoped resources when CR is deleted", func(ctx SpecContext) {
			nfsKey := types.NamespacedName{Name: "e2e-nfs", Namespace: e2eNamespace}

			nfs := &cachev1alpha1.NFSProvisioner{}
			Expect(k8sClient.Get(ctx, nfsKey, nfs)).To(Succeed())
			Expect(k8sClient.Delete(ctx, nfs)).To(Succeed())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, nfsKey, &cachev1alpha1.NFSProvisioner{})
				return errors.IsNotFound(err)
			}).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(BeTrue())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: defaults.ClusterRole}, &rbacv1.ClusterRole{})
				return errors.IsNotFound(err)
			}).WithContext(ctx).WithTimeout(30 * time.Second).WithPolling(interval).Should(BeTrue())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: defaults.ClusterRoleBinding}, &rbacv1.ClusterRoleBinding{})
				return errors.IsNotFound(err)
			}).WithContext(ctx).WithTimeout(30 * time.Second).WithPolling(interval).Should(BeTrue())
		}, SpecTimeout(timeout))
	})
})

// ─── scForNFSPvc mode (StorageClass-backed) ───────────────────────────

const scNamespace = "e2e-nfs-sc-test"

var _ = Describe("NFS Provisioner E2E - scForNFSPvc mode", Ordered, func() {
	var scDeploymentReady bool

	BeforeAll(func(ctx SpecContext) {
		By("creating test namespace")
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: scNamespace},
		}
		err := k8sClient.Create(ctx, ns)
		if err != nil && !errors.IsAlreadyExists(err) {
			Expect(err).NotTo(HaveOccurred())
		}
	}, NodeTimeout(30*time.Second))

	AfterAll(func(ctx SpecContext) {
		By("cleaning up scForNFSPvc test resources")
		nfs := &cachev1alpha1.NFSProvisioner{
			ObjectMeta: metav1.ObjectMeta{Name: "e2e-nfs-sc", Namespace: scNamespace},
		}
		_ = k8sClient.Delete(ctx, nfs)
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name: "e2e-nfs-sc", Namespace: scNamespace,
			}, &cachev1alpha1.NFSProvisioner{})
			return errors.IsNotFound(err)
		}, "30s", "1s").Should(BeTrue())

		_ = k8sClient.Delete(ctx, &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{Name: defaults.ClusterRole},
		})
		_ = k8sClient.Delete(ctx, &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: defaults.ClusterRoleBinding},
		})
		_ = k8sClient.Delete(ctx, &storagev1.StorageClass{
			ObjectMeta: metav1.ObjectMeta{Name: defaults.SCForNFSProvisioner},
		})
		_ = k8sClient.Delete(ctx, &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: "e2e-sc-pvc", Namespace: scNamespace},
		})
		_ = k8sClient.Delete(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: scNamespace},
		})
	}, NodeTimeout(60*time.Second))

	Context("create CR with scForNFSPvc (no node prep needed)", func() {
		It("should create NFS server PVC and all resources using default StorageClass", func(ctx SpecContext) {
			defaultSC := getDefaultStorageClass()

			By(fmt.Sprintf("creating NFSProvisioner CR with scForNFSPvc=%s", defaultSC))
			nfs := &cachev1alpha1.NFSProvisioner{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "e2e-nfs-sc",
					Namespace: scNamespace,
				},
				Spec: cachev1alpha1.NFSProvisionerSpec{
					SCForNFSPvc: defaultSC,
					StorageSize: "1Gi",
				},
			}
			Expect(k8sClient.Create(ctx, nfs)).To(Succeed())

			nfsKey := types.NamespacedName{Name: "e2e-nfs-sc", Namespace: scNamespace}

			By("verifying status reaches Ready")
			Eventually(func(g Gomega) {
				latest := &cachev1alpha1.NFSProvisioner{}
				g.Expect(k8sClient.Get(ctx, nfsKey, latest)).To(Succeed())
				readyCond := apimeta.FindStatusCondition(latest.Status.Conditions, reconciler.ConditionTypeReady)
				g.Expect(readyCond).NotTo(BeNil())
				g.Expect(readyCond.Status).To(Equal(metav1.ConditionTrue))
				g.Expect(latest.Status.Phase).To(Equal(reconciler.PhaseReady))
			}).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			By("verifying NFS server PVC was created with correct StorageClass")
			Eventually(func(g Gomega) {
				pvc := &corev1.PersistentVolumeClaim{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name: defaults.Pvc, Namespace: scNamespace,
				}, pvc)).To(Succeed())
				g.Expect(pvc.Spec.StorageClassName).NotTo(BeNil())
				g.Expect(*pvc.Spec.StorageClassName).To(Equal(defaultSC))
			}).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			By("verifying Deployment and Service exist")
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: defaults.Deployment, Namespace: scNamespace,
			}, &appsv1.Deployment{})).To(Succeed())

			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: defaults.Service, Namespace: scNamespace,
			}, &corev1.Service{})).To(Succeed())

			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: defaults.SCForNFSProvisioner,
			}, &storagev1.StorageClass{})).To(Succeed())
		}, SpecTimeout(timeout))
	})

	Context("NFS server readiness with PVC storage", func() {
		It("should have the NFS server pod running", func(ctx SpecContext) {
			Eventually(func(g Gomega) {
				dep := &appsv1.Deployment{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name: defaults.Deployment, Namespace: scNamespace,
				}, dep)).To(Succeed())
				g.Expect(dep.Status.AvailableReplicas).To(BeNumerically(">=", 1))
			}).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			scDeploymentReady = true
		}, SpecTimeout(timeout))
	})

	Context("NFS volume provisioning with PVC-backed server", func() {
		It("should provision a PV via NFS StorageClass", func(ctx SpecContext) {
			if !scDeploymentReady {
				Skip("NFS server not ready")
			}

			scName := defaults.SCForNFSProvisioner
			pvc := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "e2e-sc-pvc",
					Namespace: scNamespace,
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					StorageClassName: &scName,
					AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("100Mi"),
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, pvc)).To(Succeed())

			Eventually(func(g Gomega) {
				foundPVC := &corev1.PersistentVolumeClaim{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name: "e2e-sc-pvc", Namespace: scNamespace,
				}, foundPVC)).To(Succeed())
				g.Expect(foundPVC.Status.Phase).To(Equal(corev1.ClaimBound))
			}).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(Succeed())
		}, SpecTimeout(timeout))
	})
})

// ─── Helper functions ─────────────────────────────────────────────────

// getDefaultStorageClass returns the name of the default StorageClass in the cluster.
// Falls back to "standard" (Kind default) if no default is found.
func getDefaultStorageClass() string {
	out, err := exec.Command("kubectl", "get", "storageclass",
		"-o", "jsonpath={.items[?(@.metadata.annotations.storageclass\\.kubernetes\\.io/is-default-class==\"true\")].metadata.name}").CombinedOutput()
	if err == nil {
		name := strings.TrimSpace(string(out))
		if name != "" {
			// Take first if multiple
			return strings.Fields(name)[0]
		}
	}
	// Fallback
	if isOCP() {
		return "gp3-csi"
	}
	return "standard"
}

// ─── Node preparation helpers ─────────────────────────────────────────

// prepareKindNodes sets up Kind worker nodes (docker exec + kubectl label).
func prepareKindNodes() {
	targetNode := findWorkerNode()
	GinkgoWriter.Printf("Preparing Kind node %s\n", targetNode)

	out, err := exec.Command("docker", "exec", targetNode, "mkdir", "-p", "/tmp/nfs-e2e").CombinedOutput()
	Expect(err).NotTo(HaveOccurred(), "failed to mkdir on node: "+string(out))

	out, err = exec.Command("kubectl", "label", "node", targetNode,
		"app=nfs-provisioner", "--overwrite").CombinedOutput()
	Expect(err).NotTo(HaveOccurred(), "failed to label node: "+string(out))
}

// prepareOCPNodes sets up OpenShift worker nodes (oc debug + oc label).
//
// Run with: E2E_PLATFORM=ocp go test ./test/e2e/ -v -timeout 10m
func prepareOCPNodes() {
	targetNode := findWorkerNode()
	GinkgoWriter.Printf("Preparing OCP node %s\n", targetNode)

	// Create directory and set SELinux context via oc debug
	script := "chroot /host bash -c 'mkdir -p /home/core/nfs && chcon -Rvt svirt_sandbox_file_t /home/core/nfs'"
	out, err := exec.Command("oc", "debug", "node/"+targetNode, "-n", e2eNamespace, "--", "bash", "-c", script).CombinedOutput() //nolint:gosec // test helper with controlled input
	Expect(err).NotTo(HaveOccurred(), "failed to prepare OCP node: "+string(out))

	// Label the node
	out, err = exec.Command("oc", "label", "node", targetNode,
		"app=nfs-provisioner", "--overwrite").CombinedOutput()
	Expect(err).NotTo(HaveOccurred(), "failed to label node: "+string(out))
}

// findWorkerNode returns the name of the first worker node in the cluster.
func findWorkerNode() string {
	cmd := "kubectl"
	if isOCP() {
		cmd = "oc"
	}

	// Try worker nodes first (exclude control-plane)
	out, err := exec.Command(cmd, "get", "nodes",
		"-l", "!node-role.kubernetes.io/control-plane",
		"-o", "jsonpath={.items[*].metadata.name}").CombinedOutput()
	if err == nil && strings.TrimSpace(string(out)) != "" {
		return strings.Fields(strings.TrimSpace(string(out)))[0]
	}

	// OCP uses node-role.kubernetes.io/worker label
	out, err = exec.Command(cmd, "get", "nodes",
		"-l", "node-role.kubernetes.io/worker",
		"-o", "jsonpath={.items[*].metadata.name}").CombinedOutput()
	if err == nil && strings.TrimSpace(string(out)) != "" {
		return strings.Fields(strings.TrimSpace(string(out)))[0]
	}

	// Fallback: any node
	out, _ = exec.Command(cmd, "get", "nodes",
		"-o", "jsonpath={.items[*].metadata.name}").CombinedOutput()
	nodes := strings.Fields(strings.TrimSpace(string(out)))
	Expect(nodes).NotTo(BeEmpty(), "no nodes found")
	return nodes[0]
}
