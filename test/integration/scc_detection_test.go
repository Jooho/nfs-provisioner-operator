package integration

import (
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	securityv1 "github.com/openshift/api/security/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/pkg/defaults"
	"github.com/jooho/nfs-provisioner-operator/pkg/resources"
)

var _ = Describe("SCC Detection", func() {
	var (
		nfsProvisioner *cachev1alpha1.NFSProvisioner
		sccManager     *resources.SCCManager
	)

	var sccClient client.Client

	BeforeEach(func(ctx SpecContext) {
		// Create test NFSProvisioner instance
		nfsProvisioner = &cachev1alpha1.NFSProvisioner{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-nfs",
				Namespace: "default",
				UID:       "test-uid",
			},
			Spec: cachev1alpha1.NFSProvisionerSpec{
				HostPathDir: "/mnt/nfs",
				StorageSize: "10Gi",
			},
		}

		// Default sccClient to k8sClient; overridden in OpenShift context
		sccClient = k8sClient

		// Create SCC manager (will be recreated with fresh client in OpenShift context)
		baseManager := resources.NewBaseResourceManager(k8sClient, logr.Discard(), scheme.Scheme)
		sccManager = resources.NewSCCManager(baseManager)
	})

	Context("on vanilla Kubernetes (no SCC CRD)", func() {
		It("should gracefully skip SCC creation", func(ctx SpecContext) {
			// Ensure SCC CRD is NOT present
			sccCRD := &apiextensionsv1.CustomResourceDefinition{}
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name: "securitycontextconstraints.security.openshift.io",
			}, sccCRD)
			Expect(err).To(HaveOccurred()) // CRD should not exist

			// EnsureResource should succeed without creating SCC
			err = sccManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify SCC was NOT created
			scc := &securityv1.SecurityContextConstraints{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name: defaults.SecurityContextConstraints,
			}, scc)
			Expect(err).To(HaveOccurred()) // SCC should not exist
		}, SpecTimeout(30*time.Second))
	})

	Context("on OpenShift (SCC CRD present)", func() {
		var sccCRD *apiextensionsv1.CustomResourceDefinition

		BeforeEach(func(ctx SpecContext) {
			// Create SCC CRD to simulate OpenShift environment
			sccCRD = &apiextensionsv1.CustomResourceDefinition{
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
									Type:                   "object",
									XPreserveUnknownFields: boolPtr(true),
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

			// Create the CRD
			Expect(k8sClient.Create(ctx, sccCRD)).To(Succeed())

			// Wait for CRD to be established and API discovery to recognize it
			Eventually(func(g Gomega) {
				crd := &apiextensionsv1.CustomResourceDefinition{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: sccCRD.Name}, crd)).To(Succeed())
				for _, cond := range crd.Status.Conditions {
					if cond.Type == apiextensionsv1.Established {
						g.Expect(cond.Status).To(Equal(apiextensionsv1.ConditionTrue))
						return
					}
				}
				g.Expect(false).To(BeTrue(), "CRD not yet established")
			}, "10s", "500ms").Should(Succeed())

			// Create a fresh client with updated REST mapper that knows about the new CRD
			var err error
			sccClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
			Expect(err).NotTo(HaveOccurred())

			// Recreate SCC manager with the fresh client
			baseManager := resources.NewBaseResourceManager(sccClient, logr.Discard(), scheme.Scheme)
			sccManager = resources.NewSCCManager(baseManager)
		})

		AfterEach(func(ctx SpecContext) {
			// Clean up SCC CRD
			if sccCRD != nil {
				err := k8sClient.Delete(ctx, sccCRD)
				if err == nil {
					// Wait for deletion
					Eventually(func() bool {
						err := k8sClient.Get(ctx, types.NamespacedName{
							Name: sccCRD.Name,
						}, sccCRD)
						return err != nil
					}, "10s", "1s").Should(BeTrue())
				}
			}

			// Clean up SCC if exists (use sccClient which has fresh REST mapper)
			if sccClient != nil {
				scc := &securityv1.SecurityContextConstraints{}
				if err := sccClient.Get(ctx, types.NamespacedName{
					Name: defaults.SecurityContextConstraints,
				}, scc); err == nil {
					_ = sccClient.Delete(ctx, scc)
				}
			}
		})

		It("should detect SCC CRD and create SCC", func(ctx SpecContext) {
			// Verify SCC CRD is present
			foundCRD := &apiextensionsv1.CustomResourceDefinition{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: "securitycontextconstraints.security.openshift.io",
			}, foundCRD)).To(Succeed())

			// EnsureResource should create SCC
			Expect(sccManager.EnsureResource(ctx, nfsProvisioner)).To(Succeed())

			// Verify SCC was created (use sccClient with fresh REST mapper)
			scc := &securityv1.SecurityContextConstraints{}
			Expect(sccClient.Get(ctx, types.NamespacedName{
				Name: defaults.SecurityContextConstraints,
			}, scc)).To(Succeed())

			// Verify SCC has correct user
			expectedUser := "system:serviceaccount:default:" + defaults.ServiceAccount
			Expect(scc.Users).To(ContainElement(expectedUser))
		}, SpecTimeout(30*time.Second))

		It("should add user to existing SCC without duplicates", func(ctx SpecContext) {
			// Create existing SCC using sccClient (fresh REST mapper)
			existingSCC := &securityv1.SecurityContextConstraints{
				ObjectMeta: metav1.ObjectMeta{
					Name: defaults.SecurityContextConstraints,
				},
				Users: []string{"system:serviceaccount:other-namespace:other-sa"},
			}
			Expect(sccClient.Create(ctx, existingSCC)).To(Succeed())

			// First call - should add user
			err := sccManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			// Verify user was added
			scc := &securityv1.SecurityContextConstraints{}
			Expect(sccClient.Get(ctx, types.NamespacedName{
				Name: defaults.SecurityContextConstraints,
			}, scc)).To(Succeed())

			expectedUser := "system:serviceaccount:default:" + defaults.ServiceAccount
			Expect(scc.Users).To(ContainElement(expectedUser))
			Expect(scc.Users).To(HaveLen(2))

			// Second call - should be idempotent
			err = sccManager.EnsureResource(ctx, nfsProvisioner)
			Expect(err).NotTo(HaveOccurred())

			Expect(sccClient.Get(ctx, types.NamespacedName{
				Name: defaults.SecurityContextConstraints,
			}, scc)).To(Succeed())
			Expect(scc.Users).To(HaveLen(2)) // Still 2, no duplicates
		}, SpecTimeout(30*time.Second))
	})
})

func boolPtr(b bool) *bool {
	return &b
}
