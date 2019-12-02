package ups

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1/types"

	upsv1alpha1 "github.com/aerogear/unifiedpush-operator/pkg/apis/push/v1alpha1"
	croUtil "github.com/integr8ly/cloud-resource-operator/pkg/resources"
	v1alpha12 "github.com/integr8ly/integreatly-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/monitoring"
	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/pkg/errors"

	pkgerr "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultInstallationNamespace = "ups"
	defaultUpsName               = "ups"
	defaultSubscriptionName      = "integreatly-unifiedpush"
	defaultRoutename             = defaultUpsName + "-unifiedpush-proxy"
	manifestPackage              = "integreatly-unifiedpush"
	tier                         = "production"
)

type Reconciler struct {
	Config        *config.Ups
	ConfigManager config.ConfigReadWriter
	mpm           marketplace.MarketplaceInterface
	logger        *logrus.Entry
	*resources.Reconciler
}

func NewReconciler(configManager config.ConfigReadWriter, instance *v1alpha1.Installation, mpm marketplace.MarketplaceInterface) (*Reconciler, error) {
	upsConfig, err := configManager.ReadUps()
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve ups config")
	}

	if upsConfig.GetNamespace() == "" {
		upsConfig.SetNamespace(instance.Spec.NamespacePrefix + defaultInstallationNamespace)
	}

	logger := logrus.NewEntry(logrus.StandardLogger())

	return &Reconciler{
		ConfigManager: configManager,
		Config:        upsConfig,
		mpm:           mpm,
		logger:        logger,
		Reconciler:    resources.NewReconciler(mpm),
	}, nil
}

func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "unifiedpush-operator",
			Namespace: ns,
		},
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, inst *v1alpha1.Installation, product *v1alpha1.InstallationProductStatus, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	logrus.Infof("Reconciling %s", defaultUpsName)

	phase, err := r.ReconcileFinalizer(ctx, serverClient, inst, string(r.Config.GetProductName()), func() (v1alpha1.StatusPhase, error) {
		phase, err := resources.RemoveNamespace(ctx, inst, serverClient, r.Config.GetNamespace())
		if err != nil || phase != v1alpha1.PhaseCompleted {
			return phase, err
		}
		return v1alpha1.PhaseCompleted, nil
	})
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	ns := r.Config.GetNamespace()

	phase, err = r.ReconcileNamespace(ctx, ns, inst, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	namespace, err := resources.GetNS(ctx, ns, serverClient)
	if err != nil {
		return v1alpha1.PhaseFailed, err
	}

	phase, err = r.ReconcileSubscription(ctx, namespace, marketplace.Target{Pkg: defaultSubscriptionName, Namespace: ns, Channel: marketplace.IntegreatlyChannel, ManifestPackage: manifestPackage}, ns, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileComponents(ctx, inst, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileHost(ctx, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileBlackboxTargets(ctx, inst, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()
	product.OperatorVersion = r.Config.GetOperatorVersion()

	logrus.Infof("%s is successfully reconciled", defaultUpsName)

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileComponents(ctx context.Context, inst *v1alpha1.Installation, client pkgclient.Client) (v1alpha1.StatusPhase, error) {
	logrus.Info("Reconciling external postgres")
	ns := inst.Namespace

	// setup postgres custom resource
	// this will be used by the cloud resources operator to provision a postgres instance
	postgresName := fmt.Sprintf("ups-postgres-%s", inst.Name)
	postgres, err := croUtil.ReconcilePostgres(ctx, client, inst.Spec.Type, tier, postgresName, ns, postgresName, ns, func(cr metav1.Object) error {
		resources.AddOwner(cr, inst)
		return nil
	})
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to reconcile postgres request")
	}

	// wait for the postgres instance to reconcile
	if postgres.Status.Phase != types.PhaseComplete {
		return v1alpha1.PhaseAwaitingComponents, nil
	}

	// get the secret created by the cloud resources operator
	// containing postgres connection details
	connSec := &v1.Secret{}
	err = client.Get(ctx, pkgclient.ObjectKey{Name: postgres.Status.SecretRef.Name, Namespace: postgres.Status.SecretRef.Namespace}, connSec)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "failed to get postgres credential secret")
	}

	// Reconcile ups custom resource
	logrus.Info("Reconciling unified push server cr")
	cr := &upsv1alpha1.UnifiedPushServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultUpsName,
			Namespace: r.Config.GetNamespace(),
		},
		Spec: upsv1alpha1.UnifiedPushServerSpec{
			ExternalDB: true,
			Database: upsv1alpha1.UnifiedPushServerDatabase{
				Name:     string(connSec.Data["database"]),
				Password: string(connSec.Data["password"]),
				User:     string(connSec.Data["username"]),
				Host:     string(connSec.Data["host"]),
				Port:     intstr.FromString(string(connSec.Data["port"])),
			},
		},
	}

	err = client.Get(ctx, pkgclient.ObjectKey{Name: cr.Name, Namespace: cr.Namespace}, cr)
	if err != nil {
		// If the error is not an isNotFound error
		if !k8serr.IsNotFound(err) {
			return v1alpha1.PhaseFailed, err
		}

		// Otherwise create the cr
		if err := client.Create(ctx, cr); err != nil {
			return v1alpha1.PhaseFailed, pkgerr.Wrap(err, "failed to create unified push server custom resource during reconcile")
		}
	}

	// Wait till the ups cr status is complete
	if cr.Status.Phase != upsv1alpha1.PhaseReconciling {
		logrus.Info("Waiting for unified push server cr phase to complete")
		return v1alpha1.PhaseInProgress, nil
	}

	logrus.Info("Successfully reconciled unified push server custom resource")
	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileHost(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	// Setting host on config to exposed route
	logrus.Info("Setting unified push server config host")
	upsRoute := &routev1.Route{}
	err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: defaultRoutename, Namespace: r.Config.GetNamespace()}, upsRoute)
	if err != nil {
		return v1alpha1.PhaseFailed, pkgerr.Wrapf(err, "failed to get route for unified push server")
	}

	r.Config.SetHost("https://" + upsRoute.Spec.Host)
	err = r.ConfigManager.WriteConfig(r.Config)
	if err != nil {
		return v1alpha1.PhaseFailed, pkgerr.Wrap(err, "could not update unified push server config")
	}

	logrus.Info("Successfully set unified push server host")

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileBlackboxTargets(ctx context.Context, inst *v1alpha1.Installation, client pkgclient.Client) (v1alpha1.StatusPhase, error) {
	cfg, err := r.ConfigManager.ReadMonitoring()
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "error reading monitoring config")
	}

	err = monitoring.CreateBlackboxTarget("integreatly-ups", v1alpha12.BlackboxtargetData{
		Url:     r.Config.GetHost(),
		Service: "ups-ui",
	}, ctx, cfg, inst, client)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "error creating ups blackbox target")
	}

	return v1alpha1.PhaseCompleted, nil
}
