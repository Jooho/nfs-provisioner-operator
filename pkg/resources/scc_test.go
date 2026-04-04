package resources

import (
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	securityv1 "github.com/openshift/api/security/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/pkg/defaults"
)

var _ = Describe("SCCManager", func() {
	var (
		sccManager     *SCCManager
		nfsProvisioner *cachev1alpha1.NFSProvisioner
		testScheme     *runtime.Scheme
	)

	BeforeEach(func(ctx SpecContext) {
		// Create scheme
		testScheme = runtime.NewScheme()
		Expect(securityv1.AddToScheme(testScheme)).To(Succeed())
		Expect(apiextensionsv1.AddToScheme(testScheme)).To(Succeed())
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
	})

	Describe("EnsureResource on OpenShift", func() {
		BeforeEach(func(ctx SpecContext) {
			// Create SCC CRD to simulate OpenShift environment
			sccCRD := &apiextensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "securitycontextconstraints.security.openshift.io",
				},
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Group: "security.openshift.io",
					Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
						{
							Name:    "v1",
							Served:  true,
							Storage: true,
							Schema: &apiextensionsv1.CustomResourceValidation{
								OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
									Type: "object",
								},
							},
						},
					},
					Scope: apiextensionsv1.ClusterScoped,
					Names: apiextensionsv1.CustomResourceDefinitionNames{
						Plural: "securitycontextconstraints",
						Kind:   "SecurityContextConstraints",
					},
				},
			}

			// Create fake client with SCC CRD
			fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(sccCRD).Build()

			// Create SCC manager
			baseManager := NewBaseResourceManager(fakeClient, logr.Discard(), testScheme)
			sccManager = NewSCCManager(baseManager)
		})

		It("should create SCC when it doesn't exist", func(ctx SpecContext) {
			err := sccManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify SCC was created
			scc := &securityv1.SecurityContextConstraints{}
			err = sccManager.Client.Get(ctx, types.NamespacedName{
				Name: defaults.SecurityContextConstraints,
			}, scc)
			Expect(err).NotTo(HaveOccurred())

			// Verify SCC metadata
			Expect(scc.Name).To(Equal(defaults.SecurityContextConstraints))

			// Verify user was added
			expectedUser := "system:serviceaccount:test-namespace:" + defaults.ServiceAccount
			Expect(scc.Users).To(ContainElement(expectedUser))

			// Verify SCC permissions
			Expect(scc.AllowHostDirVolumePlugin).To(BeTrue())
			Expect(scc.AllowPrivilegedContainer).To(BeFalse())
		})

		It("should add user to existing SCC without duplicates", func(ctx SpecContext) {
			// Create existing SCC without our user
			existingSCC := &securityv1.SecurityContextConstraints{
				ObjectMeta: metav1.ObjectMeta{
					Name: defaults.SecurityContextConstraints,
				},
				Users: []string{"system:serviceaccount:other-namespace:other-sa"},
			}
			Expect(sccManager.Client.Create(ctx, existingSCC)).To(Succeed())

			// First call - should add user
			err := sccManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify user was added
			scc := &securityv1.SecurityContextConstraints{}
			err = sccManager.Client.Get(ctx, types.NamespacedName{
				Name: defaults.SecurityContextConstraints,
			}, scc)
			Expect(err).NotTo(HaveOccurred())

			expectedUser := "system:serviceaccount:test-namespace:" + defaults.ServiceAccount
			Expect(scc.Users).To(ContainElement(expectedUser))
			Expect(scc.Users).To(ContainElement("system:serviceaccount:other-namespace:other-sa"))
			Expect(scc.Users).To(HaveLen(2))

			// Second call - should be idempotent (no duplicate user)
			err = sccManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			err = sccManager.Client.Get(ctx, types.NamespacedName{
				Name: defaults.SecurityContextConstraints,
			}, scc)
			Expect(err).NotTo(HaveOccurred())
			Expect(scc.Users).To(HaveLen(2)) // Still 2, no duplicates
		})

		It("should not add user if already exists", func(ctx SpecContext) {
			expectedUser := "system:serviceaccount:test-namespace:" + defaults.ServiceAccount

			// Create existing SCC with our user already present
			existingSCC := &securityv1.SecurityContextConstraints{
				ObjectMeta: metav1.ObjectMeta{
					Name: defaults.SecurityContextConstraints,
				},
				Users: []string{expectedUser},
			}
			Expect(sccManager.Client.Create(ctx, existingSCC)).To(Succeed())

			err := sccManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify user count hasn't changed
			scc := &securityv1.SecurityContextConstraints{}
			err = sccManager.Client.Get(ctx, types.NamespacedName{
				Name: defaults.SecurityContextConstraints,
			}, scc)
			Expect(err).NotTo(HaveOccurred())
			Expect(scc.Users).To(HaveLen(1))
			Expect(scc.Users[0]).To(Equal(expectedUser))
		})
	})

	Describe("EnsureResource on vanilla Kubernetes", func() {
		BeforeEach(func(ctx SpecContext) {
			// Create fake client WITHOUT SCC CRD (vanilla Kubernetes)
			fakeClient := fake.NewClientBuilder().WithScheme(testScheme).Build()

			// Create SCC manager
			baseManager := NewBaseResourceManager(fakeClient, logr.Discard(), testScheme)
			sccManager = NewSCCManager(baseManager)
		})

		It("should skip SCC creation when CRD is not available", func(ctx SpecContext) {
			err := sccManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify SCC was NOT created
			scc := &securityv1.SecurityContextConstraints{}
			err = sccManager.Client.Get(ctx, types.NamespacedName{
				Name: defaults.SecurityContextConstraints,
			}, scc)
			Expect(err).To(HaveOccurred()) // Should error because SCC doesn't exist
		})
	})

	Describe("GetResourceName", func() {
		BeforeEach(func(ctx SpecContext) {
			fakeClient := fake.NewClientBuilder().WithScheme(testScheme).Build()
			baseManager := NewBaseResourceManager(fakeClient, logr.Discard(), testScheme)
			sccManager = NewSCCManager(baseManager)
		})

		It("should return correct resource name", func(ctx SpecContext) {
			Expect(sccManager.GetResourceName()).To(Equal("SecurityContextConstraints"))
		})
	})
})
