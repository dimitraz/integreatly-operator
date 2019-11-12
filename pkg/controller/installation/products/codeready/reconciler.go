package codeready

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	chev1 "github.com/eclipse/che-operator/pkg/apis/org/v1"
	crov1 "github.com/integr8ly/cloud-resource-operator/pkg/apis/integreatly/v1alpha1"
	croUtil "github.com/integr8ly/cloud-resource-operator/pkg/resources"

	keycloakv1 "github.com/integr8ly/integreatly-operator/pkg/apis/aerogear/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/marketplace"
	"github.com/integr8ly/integreatly-operator/pkg/controller/installation/products/config"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/ownerutil"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultInstallationNamespace = "codeready-workspaces"
	defaultClientName            = "che-client"
	defaultCheClusterName        = "integreatly-cluster"
	defaultSubscriptionName      = "integreatly-codeready-workspaces"
	tier                         = "production"
)

type Reconciler struct {
	Config        *config.CodeReady
	ConfigManager config.ConfigReadWriter
	mpm           marketplace.MarketplaceInterface
	logger        *logrus.Entry
	*resources.Reconciler
}

func NewReconciler(configManager config.ConfigReadWriter, instance *v1alpha1.Installation, mpm marketplace.MarketplaceInterface) (*Reconciler, error) {
	config, err := configManager.ReadCodeReady()
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve che config")
	}
	if config.GetNamespace() == "" {
		config.SetNamespace(instance.Spec.NamespacePrefix + defaultInstallationNamespace)
	}

	logger := logrus.NewEntry(logrus.StandardLogger())

	return &Reconciler{
		ConfigManager: configManager,
		Config:        config,
		mpm:           mpm,
		logger:        logger,
		Reconciler:    resources.NewReconciler(mpm),
	}, nil
}

func (r *Reconciler) GetPreflightObject(ns string) runtime.Object {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "codeready",
			Namespace: ns,
		},
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, inst *v1alpha1.Installation, product *v1alpha1.InstallationProductStatus, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	phase, err := r.ReconcileNamespace(ctx, r.Config.GetNamespace(), inst, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}
	version, err := resources.NewVersion(v1alpha1.OperatorVersionCodeReadyWorkspaces)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "invalid version number for codeready")
	}
	phase, err = r.ReconcileSubscription(ctx, inst, marketplace.Target{Pkg: defaultSubscriptionName, Channel: marketplace.IntegreatlyChannel, Namespace: r.Config.GetNamespace()}, serverClient, version)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileCheCluster(ctx, inst, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}
	phase, err = r.reconcileKeycloakClient(ctx, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	phase, err = r.reconcileBackups(ctx, inst, serverClient)
	if err != nil || phase != v1alpha1.PhaseCompleted {
		return phase, err
	}

	product.Host = r.Config.GetHost()
	product.Version = r.Config.GetProductVersion()

	r.logger.Infof("%s has reconciled successfully", r.Config.GetProductName())
	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileBackups(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	backupConfig := resources.BackupConfig{
		Namespace:     r.Config.GetNamespace(),
		Name:          "codeready",
		BackendSecret: resources.BackupSecretLocation{Name: r.Config.GetBackendSecretName(), Namespace: r.ConfigManager.GetOperatorNamespace()},
		Components: []resources.BackupComponent{
			{
				Name:     "codeready-postgres-backup",
				Type:     "postgres",
				Secret:   resources.BackupSecretLocation{Name: r.Config.GetPostgresBackupSecretName(), Namespace: r.Config.GetNamespace()},
				Schedule: r.Config.GetBackupSchedule(),
			},
			{
				Name:     "codeready-pv-backup",
				Type:     "codeready_pv",
				Schedule: r.Config.GetBackupSchedule(),
			},
		},
	}
	err := r.reconcilePostgresSecret(ctx, serverClient)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrapf(err, "failed to reconcile postgres component backup secret")
	}
	err = resources.ReconcileBackup(ctx, serverClient, inst, backupConfig)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrapf(err, "failed to create backups for codeready")
	}

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcilePostgresSecret(ctx context.Context, serverClient pkgclient.Client) error {
	//get values from deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.Config.GetNamespace(),
			Name:      "postgres",
		},
	}
	err := serverClient.Get(ctx, pkgclient.ObjectKey{Namespace: deployment.Namespace, Name: deployment.Name}, deployment)
	if err != nil {
		return errors.Wrapf(err, "could not get postgres deployment to reconcile component backup secret")
	}

	postgresqlSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.Config.GetPostgresBackupSecretName(),
			Namespace: r.Config.GetNamespace(),
		},
		Data: map[string][]byte{
			"POSTGRES_HOST": []byte("postgres." + r.Config.GetNamespace() + ".svc"),
		},
	}

	for _, env := range deployment.Spec.Template.Spec.Containers[0].Env {
		switch env.Name {
		case "POSTGRESQL_USER":
			postgresqlSecret.Data["POSTGRES_USERNAME"] = []byte(env.Value)
		case "POSTGRESQL_PASSWORD":
			postgresqlSecret.Data["POSTGRES_PASSWORD"] = []byte(env.Value)
		case "POSTGRESQL_DATABASE":
			postgresqlSecret.Data["POSTGRES_DATABASE"] = []byte(env.Value)
		case "POSTGRESQL_ADMIN_PASSWORD":
			postgresqlSecret.Data["POSTGRES_ADMIN_PASSWORD"] = []byte(env.Value)
		}
	}

	err = resources.CreateOrUpdate(ctx, serverClient, postgresqlSecret)
	if err != nil {
		return errors.Wrapf(err, "error reconciling postgres component backup secret")
	}
	logrus.Infof("codeready postgres component backup secret successfully reconciled")

	return nil
}

func (r *Reconciler) reconcileCheCluster(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	kcConfig, err := r.ConfigManager.ReadRHSSO()
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "could not retrieve keycloak config")
	}
	if err = kcConfig.Validate(); err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "keycloak config is not valid")
	}

	r.logger.Infof("creating required custom resources in namespace: %s", r.Config.GetNamespace())

	kcRealm := &keycloakv1.KeycloakRealm{}
	key := pkgclient.ObjectKey{Name: kcConfig.GetRealm(), Namespace: kcConfig.GetNamespace()}
	err = serverClient.Get(ctx, key, kcRealm)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, fmt.Sprintf("could not retrieve: %+v", key))
	}

	cheCluster := &chev1.CheCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultCheClusterName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	err = serverClient.Get(ctx, pkgclient.ObjectKey{Name: defaultCheClusterName, Namespace: r.Config.GetNamespace()}, cheCluster)
	if err != nil {
		if !k8serr.IsNotFound(err) {
			return v1alpha1.PhaseFailed, errors.Wrap(err, fmt.Sprintf("could not retrieve checluster custom resource in namespace: %s", r.Config.GetNamespace()))
		}
		cheCluster, err := r.createCheCluster(ctx, kcConfig, kcRealm, inst, serverClient)
		if err != nil {
			return v1alpha1.PhaseFailed, errors.Wrap(err, fmt.Sprintf("could not create checluster custom resource in namespace: %s", r.Config.GetNamespace()))
		}
		// che cluster hasn't reconciled yet
		if cheCluster == nil {
			return v1alpha1.PhaseAwaitingComponents, nil
		}
		return v1alpha1.PhaseInProgress, err
	}

	// check cr values
	if cheCluster.Spec.Auth.ExternalKeycloak &&
		!cheCluster.Spec.Auth.OpenShiftOauth &&
		cheCluster.Spec.Auth.KeycloakURL == kcConfig.GetHost() &&
		cheCluster.Spec.Auth.KeycloakRealm == kcConfig.GetRealm() &&
		cheCluster.Spec.Auth.KeycloakClientId == defaultClientName {
		logrus.Debug("skipping checluster custom resource update as all values are correct")
		return v1alpha1.PhaseCompleted, nil
	}

	// update cr values
	cheCluster.Spec.Auth.ExternalKeycloak = true
	cheCluster.Spec.Auth.OpenShiftOauth = false
	cheCluster.Spec.Auth.KeycloakURL = kcConfig.GetHost()
	cheCluster.Spec.Auth.KeycloakRealm = kcRealm.Name
	cheCluster.Spec.Auth.KeycloakClientId = defaultClientName
	if err = serverClient.Update(ctx, cheCluster); err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, fmt.Sprintf("could not update checluster custom resource in namespace: %s", r.Config.GetNamespace()))
	}

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) handleProgressPhase(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	r.logger.Info("checking that checluster custom resource is marked as available")

	// retrive the checluster so we can use its URL for redirect and web origins in the keycloak client
	cheCluster := &chev1.CheCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultCheClusterName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: defaultCheClusterName, Namespace: r.Config.GetNamespace()}, cheCluster)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "could not retrieve checluster for keycloak client update")
	}
	if cheCluster.Status.CheClusterRunning != "Available" {
		return v1alpha1.PhaseInProgress, nil
	}

	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) reconcileKeycloakClient(ctx context.Context, serverClient pkgclient.Client) (v1alpha1.StatusPhase, error) {
	r.logger.Infof("checking keycloak client exists for che")
	kcConfig, err := r.ConfigManager.ReadRHSSO()
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "could not retrieve keycloak config")
	}
	if err = kcConfig.Validate(); err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "keycloak config is not valid")
	}

	// retrive the checluster so we can use its URL for redirect and web origins in the keycloak client
	cheCluster := &chev1.CheCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultCheClusterName,
			Namespace: r.Config.GetNamespace(),
		},
	}
	err = serverClient.Get(ctx, pkgclient.ObjectKey{Name: defaultCheClusterName, Namespace: r.Config.GetNamespace()}, cheCluster)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "could not retrieve checluster for keycloak client update")
	}

	cheURL := cheCluster.Status.CheURL
	if cheURL == "" {
		//still waiting for the Che URL, so exit codeready reconciling now and try again
		return v1alpha1.PhaseInProgress, nil
	}

	if r.Config.GetHost() != cheURL {
		r.Config.SetHost(cheURL)
		if err = r.ConfigManager.WriteConfig(r.Config); err != nil {
			return v1alpha1.PhaseFailed, errors.Wrap(err, "could not write che configuration")
		}
	}

	// retrieve the sso config so we can find the keycloakrealm custom resource to update
	kcRealm := &keycloakv1.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kcConfig.GetRealm(),
			Namespace: kcConfig.GetNamespace(),
		},
	}
	err = serverClient.Get(ctx, pkgclient.ObjectKey{Name: kcConfig.GetRealm(), Namespace: kcConfig.GetNamespace()}, kcRealm)
	if err != nil {
		return v1alpha1.PhaseFailed, errors.Wrap(err, "could not retrieve keycloakrealm for keycloak client update")
	}

	// Create a che client that can be used in keycloak for che to login with
	if !keycloakv1.ContainsClient(kcRealm.Spec.Clients, defaultClientName) {
		r.logger.Infof("creating che client, %s, in keycloak", defaultClientName)
		kcRealm.Spec.Clients = append(kcRealm.Spec.Clients, &keycloakv1.KeycloakClient{
			KeycloakApiClient: &keycloakv1.KeycloakApiClient{
				ID:                        defaultClientName,
				ClientID:                  defaultClientName,
				ClientAuthenticatorType:   "client-secret",
				Enabled:                   true,
				PublicClient:              true,
				DirectAccessGrantsEnabled: true,
				RedirectUris:              []string{cheURL, fmt.Sprintf("%s/*", cheURL)},
				WebOrigins:                []string{cheURL, fmt.Sprintf("%s/*", cheURL)},
				StandardFlowEnabled:       true,
				RootURL:                   cheURL,
				FullScopeAllowed:          true,
				Access: map[string]bool{
					"view":      true,
					"configure": true,
					"manage":    true,
				},
				ProtocolMappers: []keycloakv1.KeycloakProtocolMapper{
					{
						Name:            "given name",
						Protocol:        "openid-connect",
						ProtocolMapper:  "oidc-usermodel-property-mapper",
						ConsentRequired: true,
						ConsentText:     "${givenName}",
						Config: map[string]string{
							"userinfo.token.claim": "true",
							"user.attribute":       "firstName",
							"id.token.claim":       "true",
							"access.token.claim":   "true",
							"claim.name":           "given_name",
							"jsonType.label":       "String",
						},
					},
					{
						Name:            "full name",
						Protocol:        "openid-connect",
						ProtocolMapper:  "oidc-full-name-mapper",
						ConsentRequired: true,
						ConsentText:     "${fullName}",
						Config: map[string]string{
							"id.token.claim":       "true",
							"access.token.claim":   "true",
							"userinfo.token.claim": "true",
						},
					},
					{
						Name:            "family name",
						Protocol:        "openid-connect",
						ProtocolMapper:  "oidc-usermodel-property-mapper",
						ConsentRequired: true,
						ConsentText:     "${familyName}",
						Config: map[string]string{
							"userinfo.token.claim": "true",
							"user.attribute":       "lastName",
							"id.token.claim":       "true",
							"access.token.claim":   "true",
							"claim.name":           "family_name",
							"jsonType.label":       "String",
						},
					},
					{
						Name:            "role list",
						Protocol:        "saml",
						ProtocolMapper:  "saml-role-list-mapper",
						ConsentRequired: false,
						ConsentText:     "${familyName}",
						Config: map[string]string{
							"single":               "false",
							"attribute.nameformat": "Basic",
							"attribute.name":       "Role",
						},
					},
					{
						Name:            "email",
						Protocol:        "openid-connect",
						ProtocolMapper:  "oidc-usermodel-property-mapper",
						ConsentRequired: true,
						ConsentText:     "${email}",
						Config: map[string]string{
							"userinfo.token.claim": "true",
							"user.attribute":       "email",
							"id.token.claim":       "true",
							"access.token.claim":   "true",
							"claim.name":           "email",
							"jsonType.label":       "String",
						},
					},
					{
						Name:            "username",
						Protocol:        "openid-connect",
						ProtocolMapper:  "oidc-usermodel-property-mapper",
						ConsentRequired: false,
						Config: map[string]string{
							"userinfo.token.claim": "true",
							"user.attribute":       "username",
							"id.token.claim":       "true",
							"access.token.claim":   "true",
							"claim.name":           "preferred_username",
							"jsonType.label":       "String",
						},
					},
				},
			},
		})
		if err = serverClient.Update(ctx, kcRealm); err != nil {
			return v1alpha1.PhaseFailed, errors.Wrap(err, "could not update keycloakrealm custom resource with codeready client")
		}
	}
	return v1alpha1.PhaseCompleted, nil
}

func (r *Reconciler) createCheCluster(ctx context.Context, kcCfg *config.RHSSO, kr *keycloakv1.KeycloakRealm, inst *v1alpha1.Installation, serverClient pkgclient.Client) (*chev1.CheCluster, error) {
	selfSignedCerts := inst.Spec.SelfSignedCerts
	cheDb := chev1.CheClusterSpecDB{
		ExternalDB:            false,
		ChePostgresDb:         "",
		ChePostgresPassword:   "",
		ChePostgresPort:       "",
		ChePostgresUser:       "",
		ChePostgresDBHostname: "",
	}

	// setup external postgres db if UseExternalResource set to true
	if inst.Spec.UseExternalResources {
		cheClusterExternalPostgres, err := r.reconcileExternalPostgres(ctx, inst, serverClient)
		if err != nil {
			return nil, err
		}
		if cheClusterExternalPostgres == nil {
			return nil, nil
		}
		cheDb = cheClusterExternalPostgres.Spec.Database
	}

	cheCluster := &chev1.CheCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultCheClusterName,
			Namespace: r.Config.GetNamespace(),
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: fmt.Sprintf(
				"%s/%s",
				chev1.SchemeGroupVersion.Group,
				chev1.SchemeGroupVersion.Version,
			),
			Kind: "CheCluster",
		},
		Spec: chev1.CheClusterSpec{
			Server: chev1.CheClusterSpecServer{
				CheFlavor:      "codeready",
				TlsSupport:     true,
				SelfSignedCert: selfSignedCerts,
			},
			Database: cheDb,
			Auth: chev1.CheClusterSpecAuth{
				OpenShiftOauth:   false,
				ExternalKeycloak: true,
				KeycloakURL:      kcCfg.GetHost(),
				KeycloakRealm:    kr.Name,
				KeycloakClientId: defaultClientName,
			},
			Storage: chev1.CheClusterSpecStorage{
				PvcStrategy:       "per-workspace",
				PvcClaimSize:      "1Gi",
				PreCreateSubPaths: true,
			},
		},
	}

	ownerutil.EnsureOwner(cheCluster, inst)
	if err := serverClient.Create(ctx, cheCluster); err != nil {
		return nil, errors.Wrap(err, "failed to create che cluster resource")
	}
	return cheCluster, nil
}

func (r *Reconciler) reconcileExternalPostgres(ctx context.Context, inst *v1alpha1.Installation, serverClient pkgclient.Client) (*chev1.CheCluster, error) {
	ns := inst.Namespace

	// setup postgres cr for the cloud resource operator
	postgresName := fmt.Sprintf("codeready-postgres-%s", inst.Name)
	postgres, err := croUtil.ReconcilePostgres(ctx, serverClient, inst.Spec.Type, tier, postgresName, ns, postgresName, ns, func(cr metav1.Object) error {
		resources.PrepareObject(cr, inst)
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to reconcile postgres request")
	}

	// phase is not complete, wait
	if postgres.Status.Phase != crov1.PhaseComplete {
		return nil, nil
	}

	// get the secret containing postgres connection details
	secRef := postgres.Status.SecretRef
	credSec := &v1.Secret{}
	if err := serverClient.Get(ctx, pkgclient.ObjectKey{Name: secRef.Name, Namespace: secRef.Namespace}, credSec); err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve credential secret for %s", postgresName)
	}

	// set the values on the object and hope it doesn't overwrite the rest of object
	cheCluster := &chev1.CheCluster{
		Spec: chev1.CheClusterSpec{
			Database: chev1.CheClusterSpecDB{
				ExternalDB:            true,
				ChePostgresDb:         string(credSec.Data["database"]),
				ChePostgresPassword:   string(credSec.Data["password"]),
				ChePostgresPort:       string(credSec.Data["port"]),
				ChePostgresUser:       string(credSec.Data["username"]),
				ChePostgresDBHostname: string(credSec.Data["host"]),
			},
		},
	}
	return cheCluster, nil
}
