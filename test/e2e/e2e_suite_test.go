package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	securityv1 "github.com/openshift/api/security/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
	"github.com/jooho/nfs-provisioner-operator/controllers"
	pkgreconciler "github.com/jooho/nfs-provisioner-operator/pkg/reconciler"
	"github.com/jooho/nfs-provisioner-operator/pkg/resources"
	"github.com/jooho/nfs-provisioner-operator/pkg/validation"
)

// E2E tests run against a real Kubernetes cluster (Kind).
//
// Prerequisites:
//   - Kind cluster running: kind create cluster
//   - CRDs installed: kubectl apply -f config/crd/bases/
//
// Run:
//   export PATH=~/dev/lang/go/bin:$PATH
//   go test ./test/e2e/ -v -timeout 10m

var (
	k8sClient client.Client
	ctx       context.Context
	cancel    context.CancelFunc
	testLog   logr.Logger
	scheme    = apiruntime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(cachev1alpha1.AddToScheme(scheme))
	utilruntime.Must(securityv1.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
}

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Suite")
}

var _ = BeforeSuite(func(suiteCtx SpecContext) {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	testLog = logf.Log.WithName("e2e-test")

	ctx, cancel = context.WithCancel(context.TODO())

	By("connecting to the cluster")
	cfg, err := config.GetConfig()
	Expect(err).NotTo(HaveOccurred(), "Failed to get kubeconfig - is a cluster running?")

	// Direct client for test assertions
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())

	By("starting the controller manager in-process")
	mgr, err := manager.New(cfg, manager.Options{
		Scheme:  scheme,
		Metrics: server.Options{BindAddress: "0"}, // disable metrics to avoid port conflicts
	})
	Expect(err).NotTo(HaveOccurred())

	// Wire up controller
	mgrClient := mgr.GetClient()
	mgrScheme := mgr.GetScheme()
	logger := ctrl.Log.WithName("controllers").WithName("NFSProvisioner")

	v := validation.NewValidator()
	base := resources.BaseResourceManager{
		Client: mgrClient,
		Scheme: mgrScheme,
		Log:    ctrl.Log.WithName("resources"),
	}

	resourceManagers := []resources.ResourceManager{
		resources.NewServiceAccountManager(base),
		resources.NewRBACManager(base),
		resources.NewSCCManager(base),
		resources.NewPVCManager(base),
		resources.NewDeploymentManager(base),
		resources.NewServiceManager(base),
		resources.NewStorageClassManager(base),
	}

	reconciler := pkgreconciler.NewReconciler(mgrClient, v, resourceManagers, logger)

	err = (&controllers.NFSProvisionerReconciler{
		Client:     mgrClient,
		Log:        logger,
		Scheme:     mgrScheme,
		Reconciler: reconciler,
	}).SetupWithManager(mgr)
	Expect(err).NotTo(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		Expect(mgr.Start(ctx)).To(Succeed(), "failed to run manager")
	}()
}, NodeTimeout(60*time.Second))

var _ = AfterSuite(func() {
	if cancel != nil {
		cancel()
	}
})
