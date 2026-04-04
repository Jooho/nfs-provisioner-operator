package builder_test

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/pkg/builder"
	"github.com/jooho/nfs-provisioner-operator/pkg/defaults"
)

func TestBuilder(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Builder Suite")
}

var _ = Describe("Builder", func() {
	var nfs *cachev1alpha1.NFSProvisioner

	BeforeEach(func(ctx SpecContext) {
		nfs = &cachev1alpha1.NFSProvisioner{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-nfs",
				Namespace: "test-namespace",
			},
			Spec: cachev1alpha1.NFSProvisionerSpec{
				HostPathDir:         "/mnt/nfs",
				StorageSize:         "20Gi",
				SCForNFSProvisioner: "nfs",
				NFSImageConfiguration: &cachev1alpha1.ImageConfiguration{
					Image:           ptr.To("test-image:v1"),
					ImagePullPolicy: ptr.To(corev1.PullAlways),
				},
			},
		}
	})

	Describe("BuildDeployment", func() {
		Context("with hostPathDir storage", func() {
			It("should create deployment with hostPath volume", func(ctx SpecContext) {
				deployment := builder.BuildDeployment(nfs)

				Expect(deployment).NotTo(BeNil())
				Expect(deployment.Name).To(Equal(defaults.Deployment))
				Expect(deployment.Namespace).To(Equal("test-namespace"))

				// Verify hostPath volume
				volumes := deployment.Spec.Template.Spec.Volumes
				Expect(volumes).To(HaveLen(1))
				Expect(volumes[0].Name).To(Equal("export-volume"))
				Expect(volumes[0].HostPath).NotTo(BeNil())
				Expect(volumes[0].HostPath.Path).To(Equal("/mnt/nfs"))

				// Verify container image
				containers := deployment.Spec.Template.Spec.Containers
				Expect(containers).To(HaveLen(1))
				Expect(containers[0].Image).To(Equal("test-image:v1"))
				Expect(containers[0].ImagePullPolicy).To(Equal(corev1.PullAlways))
			}, SpecTimeout(specTimeout))
		})

		Context("with PVC storage", func() {
			It("should create deployment with PVC volume", func(ctx SpecContext) {
				nfs.Spec.HostPathDir = ""
				nfs.Spec.SCForNFSPvc = "local-storage"

				deployment := builder.BuildDeployment(nfs)

				Expect(deployment).NotTo(BeNil())

				// Verify PVC volume
				volumes := deployment.Spec.Template.Spec.Volumes
				Expect(volumes).To(HaveLen(1))
				Expect(volumes[0].Name).To(Equal("export-volume"))
				Expect(volumes[0].PersistentVolumeClaim).NotTo(BeNil())
				Expect(volumes[0].PersistentVolumeClaim.ClaimName).To(Equal(defaults.Pvc))
			}, SpecTimeout(specTimeout))
		})
	})

	Describe("BuildService", func() {
		It("should create service with correct ports", func(ctx SpecContext) {
			service := builder.BuildService(nfs)

			Expect(service).NotTo(BeNil())
			Expect(service.Name).To(Equal(defaults.Service))
			Expect(service.Namespace).To(Equal("test-namespace"))

			// Verify service ports
			ports := service.Spec.Ports
			Expect(ports).To(HaveLen(12))

			// Verify we have expected NFS ports
			var hasNFSPort, hasMountdPort, hasRPCPort bool
			for _, port := range ports {
				if port.Name == "nfs" && port.Port == 2049 {
					hasNFSPort = true
				}
				if port.Name == "mountd" && port.Port == 20048 {
					hasMountdPort = true
				}
				if port.Name == "rpcbind" && port.Port == 111 {
					hasRPCPort = true
				}
			}
			Expect(hasNFSPort).To(BeTrue())
			Expect(hasMountdPort).To(BeTrue())
			Expect(hasRPCPort).To(BeTrue())

			// Verify selector
			Expect(service.Spec.Selector).To(HaveKeyWithValue("app", defaults.Deployment))
		}, SpecTimeout(specTimeout))
	})

	Describe("BuildServiceAccount", func() {
		It("should create service account with correct metadata", func(ctx SpecContext) {
			sa := builder.BuildServiceAccount(nfs)

			Expect(sa).NotTo(BeNil())
			Expect(sa.Name).To(Equal(defaults.ServiceAccount))
			Expect(sa.Namespace).To(Equal("test-namespace"))
		}, SpecTimeout(specTimeout))
	})

	Describe("BuildPVC", func() {
		Context("when using scForNFSPvc", func() {
			It("should create PVC with correct storage class", func(ctx SpecContext) {
				nfs.Spec.HostPathDir = ""
				nfs.Spec.SCForNFSPvc = "local-storage"

				pvc := builder.BuildPVC(nfs)

				Expect(pvc).NotTo(BeNil())
				Expect(pvc.Name).To(Equal(defaults.Pvc))
				Expect(pvc.Namespace).To(Equal("test-namespace"))
				Expect(*pvc.Spec.StorageClassName).To(Equal("local-storage"))

				// Verify storage size
				storageSize := pvc.Spec.Resources.Requests.Storage()
				Expect(storageSize.String()).To(Equal("20Gi"))
			}, SpecTimeout(specTimeout))
		})

		Context("when using hostPathDir", func() {
			It("should return nil (PVC not needed)", func(ctx SpecContext) {
				pvc := builder.BuildPVC(nfs)
				Expect(pvc).To(BeNil())
			}, SpecTimeout(specTimeout))
		})

		Context("when using existing PVC", func() {
			It("should return nil (PVC not created)", func(ctx SpecContext) {
				nfs.Spec.HostPathDir = ""
				nfs.Spec.Pvc = "existing-pvc"

				pvc := builder.BuildPVC(nfs)
				Expect(pvc).To(BeNil())
			}, SpecTimeout(specTimeout))
		})
	})

	Describe("BuildStorageClass", func() {
		It("should create storage class with correct provisioner", func(ctx SpecContext) {
			sc := builder.BuildStorageClass(nfs)

			Expect(sc).NotTo(BeNil())
			Expect(sc.Name).To(Equal("nfs"))
			Expect(sc.Provisioner).To(Equal("example.com/nfs"))

			// Verify no extra parameters
			Expect(sc.Parameters).To(BeNil())
		}, SpecTimeout(specTimeout))

		It("should use custom storage class name if specified", func(ctx SpecContext) {
			nfs.Spec.SCForNFSProvisioner = "custom-nfs-sc"

			sc := builder.BuildStorageClass(nfs)

			Expect(sc).NotTo(BeNil())
			Expect(sc.Name).To(Equal("custom-nfs-sc"))
		}, SpecTimeout(specTimeout))
	})

	Describe("BuildClusterRole", func() {
		It("should create cluster role with correct permissions", func(ctx SpecContext) {
			cr := builder.BuildClusterRole()

			Expect(cr).NotTo(BeNil())
			Expect(cr.Name).To(Equal(defaults.ClusterRole))

			// Verify rules
			rules := cr.Rules
			Expect(rules).NotTo(BeEmpty())

			// Check for persistent volume permissions
			var hasPVRule bool
			for _, rule := range rules {
				if contains(rule.Resources, "persistentvolumes") {
					hasPVRule = true
					Expect(rule.Verbs).To(ContainElements("get", "list", "watch", "create", "delete"))
				}
			}
			Expect(hasPVRule).To(BeTrue(), "should have persistentvolumes rule")
		}, SpecTimeout(specTimeout))
	})

	Describe("BuildClusterRoleBinding", func() {
		It("should create cluster role binding with correct subjects", func(ctx SpecContext) {
			crb := builder.BuildClusterRoleBinding(nfs)

			Expect(crb).NotTo(BeNil())
			Expect(crb.Name).To(Equal(defaults.ClusterRoleBinding))

			// Verify role ref
			Expect(crb.RoleRef.Name).To(Equal(defaults.ClusterRole))
			Expect(crb.RoleRef.Kind).To(Equal("ClusterRole"))

			// Verify subjects
			Expect(crb.Subjects).To(HaveLen(1))
			Expect(crb.Subjects[0].Kind).To(Equal("ServiceAccount"))
			Expect(crb.Subjects[0].Name).To(Equal(defaults.ServiceAccount))
			Expect(crb.Subjects[0].Namespace).To(Equal("test-namespace"))
		}, SpecTimeout(specTimeout))
	})

	Describe("BuildRole", func() {
		It("should create role with correct permissions", func(ctx SpecContext) {
			role := builder.BuildRole(nfs)

			Expect(role).NotTo(BeNil())
			Expect(role.Name).To(Equal(defaults.Role))
			Expect(role.Namespace).To(Equal("test-namespace"))

			// Verify rules
			rules := role.Rules
			Expect(rules).NotTo(BeEmpty())

			// Check for endpoints permissions
			var hasEndpointsRule bool
			for _, rule := range rules {
				if contains(rule.Resources, "endpoints") {
					hasEndpointsRule = true
					Expect(rule.Verbs).To(ContainElements("get", "list", "watch", "create", "update", "delete"))
				}
			}
			Expect(hasEndpointsRule).To(BeTrue(), "should have endpoints rule")
		}, SpecTimeout(specTimeout))
	})

	Describe("BuildRoleBinding", func() {
		It("should create role binding with correct subjects", func(ctx SpecContext) {
			rb := builder.BuildRoleBinding(nfs)

			Expect(rb).NotTo(BeNil())
			Expect(rb.Name).To(Equal(defaults.RoleBinding))
			Expect(rb.Namespace).To(Equal("test-namespace"))

			// Verify role ref
			Expect(rb.RoleRef.Name).To(Equal(defaults.Role))
			Expect(rb.RoleRef.Kind).To(Equal("Role"))

			// Verify subjects
			Expect(rb.Subjects).To(HaveLen(1))
			Expect(rb.Subjects[0].Kind).To(Equal("ServiceAccount"))
			Expect(rb.Subjects[0].Name).To(Equal(defaults.ServiceAccount))
		}, SpecTimeout(specTimeout))
	})

	Describe("BuildSCC", func() {
		It("should create SCC with correct permissions (OpenShift)", func(ctx SpecContext) {
			scc := builder.BuildSCC(nfs)

			Expect(scc).NotTo(BeNil())
			Expect(scc.Name).To(Equal(defaults.SecurityContextConstraints))

			// Verify SCC allows privileged
			Expect(scc.AllowPrivilegedContainer).To(BeFalse())

			// Verify users includes service account
			expectedUser := "system:serviceaccount:test-namespace:" + defaults.ServiceAccount
			Expect(scc.Users).To(ContainElement(expectedUser))
		}, SpecTimeout(specTimeout))
	})
})

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

const specTimeout = 5 * time.Second
