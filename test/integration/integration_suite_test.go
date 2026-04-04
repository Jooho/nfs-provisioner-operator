package integration

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	securityv1 "github.com/openshift/api/security/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/controllers"
	pkgreconciler "github.com/jooho/nfs-provisioner-operator/pkg/reconciler"
	"github.com/jooho/nfs-provisioner-operator/pkg/resources"
	"github.com/jooho/nfs-provisioner-operator/pkg/validation"
)

var (
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment
	ctx       context.Context
	cancel    context.CancelFunc
	testLog   logr.Logger
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var _ = BeforeSuite(func(suiteCtx SpecContext) {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	testLog = logf.Log.WithName("integration-test")

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	// Register schemes
	err = cachev1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = securityv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = apiextensionsv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// Create manager
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	// Create direct client for test assertions (bypasses cache)
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// Set up controller with all dependencies
	mgrClient := k8sManager.GetClient()
	mgrScheme := k8sManager.GetScheme()
	logger := ctrl.Log.WithName("controllers").WithName("NFSProvisioner")

	validator := validation.NewValidator()
	baseManager := resources.BaseResourceManager{
		Client: mgrClient,
		Scheme: mgrScheme,
		Log:    ctrl.Log.WithName("resources"),
	}

	resourceManagers := []resources.ResourceManager{
		resources.NewServiceAccountManager(baseManager),
		resources.NewRBACManager(baseManager),
		resources.NewSCCManager(baseManager),
		resources.NewPVCManager(baseManager),
		resources.NewDeploymentManager(baseManager),
		resources.NewServiceManager(baseManager),
		resources.NewStorageClassManager(baseManager),
	}

	reconciler := pkgreconciler.NewReconciler(
		mgrClient,
		validator,
		resourceManagers,
		logger,
	)

	err = (&controllers.NFSProvisionerReconciler{
		Client:     mgrClient,
		Log:        logger,
		Scheme:     mgrScheme,
		Reconciler: reconciler,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// Start the manager in background
	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()
}, NodeTimeout(60*time.Second))

var _ = AfterSuite(func() {
	if cancel != nil {
		cancel()
	}
	By("tearing down the test environment")
	if testEnv != nil {
		err := testEnv.Stop()
		Expect(err).NotTo(HaveOccurred())
	}
})
