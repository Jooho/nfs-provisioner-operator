package resources

import (
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/pkg/defaults"
)

var _ = Describe("RBACManager", func() {
	var (
		rbacManager    *RBACManager
		nfsProvisioner *cachev1alpha1.NFSProvisioner
		testScheme     *runtime.Scheme
	)

	BeforeEach(func(ctx SpecContext) {
		// Create scheme
		testScheme = runtime.NewScheme()
		Expect(rbacv1.AddToScheme(testScheme)).To(Succeed())
		Expect(cachev1alpha1.AddToScheme(testScheme)).To(Succeed())

		// Create test NFSProvisioner instance
		nfsProvisioner = &cachev1alpha1.NFSProvisioner{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-nfs",
				Namespace: "test-namespace",
				UID:       "test-uid",
			},
			Spec: cachev1alpha1.NFSProvisionerSpec{
				StorageSize: "10Gi",
			},
		}

		// Create fake client
		fakeClient := fake.NewClientBuilder().WithScheme(testScheme).Build()

		// Create RBAC manager
		baseManager := NewBaseResourceManager(fakeClient, logr.Discard(), testScheme)
		rbacManager = NewRBACManager(baseManager)
	})

	Describe("EnsureResource", func() {
		It("should create all RBAC resources", func(ctx SpecContext) {
			err := rbacManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify ClusterRole was created
			clusterRole := &rbacv1.ClusterRole{}
			err = rbacManager.Client.Get(ctx, types.NamespacedName{
				Name: defaults.ClusterRole,
			}, clusterRole)
			Expect(err).NotTo(HaveOccurred())
			Expect(clusterRole.Name).To(Equal(defaults.ClusterRole))

			// Verify ClusterRoleBinding was created
			clusterRoleBinding := &rbacv1.ClusterRoleBinding{}
			err = rbacManager.Client.Get(ctx, types.NamespacedName{
				Name: defaults.ClusterRoleBinding,
			}, clusterRoleBinding)
			Expect(err).NotTo(HaveOccurred())
			Expect(clusterRoleBinding.Name).To(Equal(defaults.ClusterRoleBinding))

			// Verify Role was created
			role := &rbacv1.Role{}
			err = rbacManager.Client.Get(ctx, types.NamespacedName{
				Name:      defaults.Role,
				Namespace: nfsProvisioner.Namespace,
			}, role)
			Expect(err).NotTo(HaveOccurred())
			Expect(role.Name).To(Equal(defaults.Role))

			// Verify RoleBinding was created
			roleBinding := &rbacv1.RoleBinding{}
			err = rbacManager.Client.Get(ctx, types.NamespacedName{
				Name:      defaults.RoleBinding,
				Namespace: nfsProvisioner.Namespace,
			}, roleBinding)
			Expect(err).NotTo(HaveOccurred())
			Expect(roleBinding.Name).To(Equal(defaults.RoleBinding))
		})

		It("should create ClusterRole with correct permissions", func(ctx SpecContext) {
			err := rbacManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify ClusterRole has required permissions
			clusterRole := &rbacv1.ClusterRole{}
			err = rbacManager.Client.Get(ctx, types.NamespacedName{
				Name: defaults.ClusterRole,
			}, clusterRole)
			Expect(err).NotTo(HaveOccurred())

			// Check for persistent volume permissions
			var hasPVRule bool
			for _, rule := range clusterRole.Rules {
				if contains(rule.Resources, "persistentvolumes") {
					hasPVRule = true
					Expect(rule.Verbs).To(ContainElements("get", "list", "watch", "create", "delete"))
				}
			}
			Expect(hasPVRule).To(BeTrue(), "should have persistentvolumes rule")
		})

		It("should create ClusterRoleBinding with correct subjects", func(ctx SpecContext) {
			err := rbacManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify ClusterRoleBinding references correct ClusterRole and ServiceAccount
			clusterRoleBinding := &rbacv1.ClusterRoleBinding{}
			err = rbacManager.Client.Get(ctx, types.NamespacedName{
				Name: defaults.ClusterRoleBinding,
			}, clusterRoleBinding)
			Expect(err).NotTo(HaveOccurred())

			// Verify RoleRef
			Expect(clusterRoleBinding.RoleRef.Name).To(Equal(defaults.ClusterRole))
			Expect(clusterRoleBinding.RoleRef.Kind).To(Equal("ClusterRole"))

			// Verify Subjects
			Expect(clusterRoleBinding.Subjects).To(HaveLen(1))
			Expect(clusterRoleBinding.Subjects[0].Kind).To(Equal("ServiceAccount"))
			Expect(clusterRoleBinding.Subjects[0].Name).To(Equal(defaults.ServiceAccount))
			Expect(clusterRoleBinding.Subjects[0].Namespace).To(Equal(nfsProvisioner.Namespace))
		})

		It("should create Role with correct permissions", func(ctx SpecContext) {
			err := rbacManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify Role has required permissions
			role := &rbacv1.Role{}
			err = rbacManager.Client.Get(ctx, types.NamespacedName{
				Name:      defaults.Role,
				Namespace: nfsProvisioner.Namespace,
			}, role)
			Expect(err).NotTo(HaveOccurred())

			// Check for endpoints permissions
			var hasEndpointsRule bool
			for _, rule := range role.Rules {
				if contains(rule.Resources, "endpoints") {
					hasEndpointsRule = true
					Expect(rule.Verbs).To(ContainElements("get", "list", "watch", "create", "update", "delete"))
				}
			}
			Expect(hasEndpointsRule).To(BeTrue(), "should have endpoints rule")
		})

		It("should create RoleBinding with correct subjects", func(ctx SpecContext) {
			err := rbacManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify RoleBinding references correct Role and ServiceAccount
			roleBinding := &rbacv1.RoleBinding{}
			err = rbacManager.Client.Get(ctx, types.NamespacedName{
				Name:      defaults.RoleBinding,
				Namespace: nfsProvisioner.Namespace,
			}, roleBinding)
			Expect(err).NotTo(HaveOccurred())

			// Verify RoleRef
			Expect(roleBinding.RoleRef.Name).To(Equal(defaults.Role))
			Expect(roleBinding.RoleRef.Kind).To(Equal("Role"))

			// Verify Subjects
			Expect(roleBinding.Subjects).To(HaveLen(1))
			Expect(roleBinding.Subjects[0].Kind).To(Equal("ServiceAccount"))
			Expect(roleBinding.Subjects[0].Name).To(Equal(defaults.ServiceAccount))
		})

		It("should create namespaced resources with owner references", func(ctx SpecContext) {
			err := rbacManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify Role has owner reference
			role := &rbacv1.Role{}
			err = rbacManager.Client.Get(ctx, types.NamespacedName{
				Name:      defaults.Role,
				Namespace: nfsProvisioner.Namespace,
			}, role)
			Expect(err).NotTo(HaveOccurred())
			Expect(role.OwnerReferences).To(HaveLen(1))
			Expect(role.OwnerReferences[0].Name).To(Equal(nfsProvisioner.Name))

			// Verify RoleBinding has owner reference
			roleBinding := &rbacv1.RoleBinding{}
			err = rbacManager.Client.Get(ctx, types.NamespacedName{
				Name:      defaults.RoleBinding,
				Namespace: nfsProvisioner.Namespace,
			}, roleBinding)
			Expect(err).NotTo(HaveOccurred())
			Expect(roleBinding.OwnerReferences).To(HaveLen(1))
			Expect(roleBinding.OwnerReferences[0].Name).To(Equal(nfsProvisioner.Name))
		})

		It("should be idempotent", func(ctx SpecContext) {
			// First call
			err := rbacManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Second call - should not error
			err = rbacManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify resources still exist
			clusterRole := &rbacv1.ClusterRole{}
			err = rbacManager.Client.Get(ctx, types.NamespacedName{Name: defaults.ClusterRole}, clusterRole)
			Expect(err).NotTo(HaveOccurred())

			role := &rbacv1.Role{}
			err = rbacManager.Client.Get(ctx, types.NamespacedName{
				Name:      defaults.Role,
				Namespace: nfsProvisioner.Namespace,
			}, role)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("GetResourceName", func() {
		It("should return correct resource name", func(ctx SpecContext) {
			Expect(rbacManager.GetResourceName()).To(Equal("RBAC"))
		})
	})
})
