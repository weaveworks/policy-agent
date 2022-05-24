package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/MagalixTechnologies/core/logger"
	"github.com/MagalixTechnologies/core/packet"
	"github.com/MagalixTechnologies/policy-core/domain"
	"github.com/MagalixTechnologies/policy-core/validation"
	"github.com/MagalixTechnologies/uuid-go"
	"github.com/fluxcd/pkg/runtime/events"
	"github.com/urfave/cli/v2"
	pacv1 "github.com/weaveworks/policy-agent/api/v1"
	"github.com/weaveworks/policy-agent/internal/admission"
	"github.com/weaveworks/policy-agent/internal/auditor"
	"github.com/weaveworks/policy-agent/internal/clients/gateway"
	"github.com/weaveworks/policy-agent/internal/clients/kube"
	"github.com/weaveworks/policy-agent/internal/entities/k8s"
	"github.com/weaveworks/policy-agent/internal/policies/crd"
	"github.com/weaveworks/policy-agent/internal/sink/filesystem"
	flux_notification "github.com/weaveworks/policy-agent/internal/sink/flux-notification"
	k8s_event "github.com/weaveworks/policy-agent/internal/sink/k8s-event"
	"github.com/weaveworks/policy-agent/internal/sink/saas"
	"github.com/weaveworks/policy-agent/pkg/log"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	SaaSSinkBatchSize   = 500
	SaaSSinkBatchExpiry = 10 * time.Second
)

// build is overriden during compilation of the binary
var build = "[runtime build]"

type Config struct {
	KubeConfigFile  string
	AccountID       string
	ClusterID       string
	WriteCompliance bool
	WebhookListen   int
	WebhookCertDir  string
	LogLevel        string
	ProbesListen    string
	EnableAdmission bool
	EnableAudit     bool

	// filesystem sink config
	FileSystemSinkFilePath string

	// kubernets event sink config
	EnableK8sEventSink bool

	// flux notification sink config
	FluxNotificationSinkAddr string

	// saas sink config
	GatewaySinkURL    string
	GatewaySinkSecret string

	MetricsAddr        string
	AuditPolicySet     string
	AdmissionPolicySet string
}

var (
	scheme = runtime.NewScheme()
)

const (
	auditControllerInterval         = 23 * time.Hour
	eventReportingController string = "policy-agent"
)

func main() {
	config := Config{}
	app := cli.NewApp()
	app.Version = "0.0.1"
	app.Name = "Policy agent"
	app.Usage = "Enforces compliance on your kubernetes cluster"
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "kube-config-file",
			Usage:       "path to kubernetes client config file",
			Destination: &config.KubeConfigFile,
			Value:       "",
			EnvVars:     []string{"AGENT_KUBE_CONFIG_FILE"},
		},
		&cli.StringFlag{
			Name:        "account-id",
			Usage:       "Account id, unique per organization",
			Destination: &config.AccountID,
			Required:    true,
			EnvVars:     []string{"AGENT_ACCOUNT_ID"},
		},
		&cli.StringFlag{
			Name:        "cluster-id",
			Usage:       "Cluster id, cluster identifier",
			Destination: &config.ClusterID,
			Required:    true,
			EnvVars:     []string{"AGENT_CLUSTER_ID"},
		},
		&cli.IntFlag{
			Name:        "webhook-listen",
			Usage:       "port for the admission webhook server to listen on",
			Destination: &config.WebhookListen,
			Value:       8443,
			EnvVars:     []string{"AGENT_WEBHOOK_LISTEN"},
		},
		&cli.StringFlag{
			Name:        "webhook-cert-dir",
			Usage:       "cert directory path for webhook server",
			Destination: &config.WebhookCertDir,
			Value:       "/certs",
			EnvVars:     []string{"AGENT_WEBHOOK_CERT_DIR"},
		},
		&cli.StringFlag{
			Name:        "probes-listen",
			Usage:       "address for the probes server to run on",
			Destination: &config.ProbesListen,
			Value:       ":9000",
			EnvVars:     []string{"AGENT_PROBES_LISTEN"},
		},
		&cli.BoolFlag{
			Name:        "write-compliance",
			Usage:       "enables writing compliance results",
			Destination: &config.WriteCompliance,
			Value:       false,
			EnvVars:     []string{"AGENT_WRITE_COMPLIANCE"},
		},
		&cli.BoolFlag{
			Name:        "admission",
			Usage:       "enables admission control",
			Destination: &config.EnableAdmission,
			Value:       false,
			EnvVars:     []string{"AGENT_ENABLE_ADMISSION"},
		},
		&cli.BoolFlag{
			Name:        "audit",
			Usage:       "enables cluster periodical audit",
			Destination: &config.EnableAudit,
			Value:       false,
			EnvVars:     []string{"AGENT_ENABLE_AUDIT"},
		},
		&cli.StringFlag{
			Name:        "log-level",
			Usage:       "app log level",
			Destination: &config.LogLevel,
			Value:       "info",
			EnvVars:     []string{"AGENT_LOG_LEVEL"},
		},
		&cli.StringFlag{
			Name:        "filesystem-sink-file-path",
			Usage:       "filesystem sink file path",
			Destination: &config.FileSystemSinkFilePath,
			EnvVars:     []string{"AGENT_FILESYSTEM_SINK_FILE_PATH"},
		},
		&cli.StringFlag{
			Name:        "flux-notification-sink-addr",
			Usage:       "flux notification sink address",
			Destination: &config.FluxNotificationSinkAddr,
			EnvVars:     []string{"AGENT_FLUX_NOTIFICATION_SINK_ADDR"},
		},
		&cli.BoolFlag{
			Name:        "enable-k8s-events-sink",
			Usage:       "enables kubernetes events sink",
			Destination: &config.EnableK8sEventSink,
			Value:       false,
			EnvVars:     []string{"AGENT_ENABLE_K8S_EVENTS_SINK"},
		},
		&cli.StringFlag{
			Name:        "gateway-sink-url",
			Usage:       "connection to the saas gateway",
			Destination: &config.GatewaySinkURL,
			EnvVars:     []string{"AGENT_GATEWAY_SINK_URL"},
		},
		&cli.StringFlag{
			Name:        "gateway-sink-secret",
			Usage:       "secret used to authenticate for the saas sink",
			Destination: &config.GatewaySinkSecret,
			EnvVars:     []string{"AGENT_GATEWAY_SINK_SECRET"},
		},
		&cli.StringFlag{
			Name:        "metrics-addr",
			Usage:       "address the metric endpoint binds to",
			Destination: &config.MetricsAddr,
			Value:       ":8080",
			EnvVars:     []string{"AGENT_METRICS_ADDR"},
		},
		&cli.StringFlag{
			Name:        "audit-policy-set",
			Usage:       "audit policy set id",
			Destination: &config.AuditPolicySet,
			EnvVars:     []string{"AGENT_AUDIT_POLICY_SET"},
		},
		&cli.StringFlag{
			Name:        "admission-policy-set",
			Usage:       "admission policy set id",
			Destination: &config.AdmissionPolicySet,
			EnvVars:     []string{"AGENT_ADMISSION_POLICY_SET"},
		},
	}

	app.Before = func(c *cli.Context) error {
		if !config.EnableAdmission && !config.EnableAudit {
			return errors.New("agent needs to be run with at least one mode of operation")
		}

		switch config.LogLevel {
		case "info":
			logger.Config(logger.InfoLevel)
		case "warn":
			logger.Config(logger.WarnLevel)
		case "debug":
			logger.Config(logger.DebugLevel)
		case "error":
			logger.Config(logger.ErrorLevel)
		default:
			return fmt.Errorf("invalid log level specified")
		}
		logger.WithGlobal("accountID", config.AccountID, "clusterID", config.ClusterID)
		return nil
	}

	app.Action = func(contextCli *cli.Context) error {
		logger.Infow("initializing Policy Agent", "build", build)
		logger.Infof("config: %+v", config)

		var kubeConfig *rest.Config
		var err error
		if config.KubeConfigFile == "" {
			kubeConfig, err = rest.InClusterConfig()
		} else {
			kubeConfig, err = clientcmd.BuildConfigFromFlags("", config.KubeConfigFile)
		}
		if err != nil {
			return fmt.Errorf("failed to load Kubernetes config: %w", err)
		}

		err = pacv1.AddToScheme(scheme)
		if err != nil {
			return fmt.Errorf("failed to add policy crd to scheme: %w", err)
		}

		lg := log.NewControllerLog(config.AccountID, config.ClusterID)

		mgr, err := ctrl.NewManager(kubeConfig, ctrl.Options{
			Scheme:                 scheme,
			MetricsBindAddress:     config.MetricsAddr,
			Port:                   config.WebhookListen,
			CertDir:                config.WebhookCertDir,
			HealthProbeBindAddress: config.ProbesListen,
			Logger:                 lg,
		})
		if err != nil {
			return fmt.Errorf("failed to initialize manager: %w", err)
		}

		err = mgr.AddHealthzCheck("liveness", healthz.Ping)
		if err != nil {
			return fmt.Errorf("failed to register liveness probe check: %w", err)
		}

		err = mgr.AddReadyzCheck("readiness", func(req *http.Request) error {
			if mgr.GetCache().WaitForCacheSync(req.Context()) {
				return nil
			} else {
				return errors.New("controller not yet ready")
			}
		})
		if err != nil {
			return fmt.Errorf("failed to register readiness probe check: %w", err)
		}

		kubeClient, err := kube.NewKubeClient(kubeConfig)
		if err != nil {
			return fmt.Errorf("init client failed: %w", err)
		}
		entitiesSources, err := k8s.GetEntitiesSources(contextCli.Context, kubeClient)
		if err != nil {
			return fmt.Errorf("initializing entities sources failed: %w", err)
		}

		auditSinks := []domain.PolicyValidationSink{}
		admissionSinks := []domain.PolicyValidationSink{}

		if config.FileSystemSinkFilePath != "" {
			logger.Infow("initializing filesystem sink ...", "file", config.FileSystemSinkFilePath)
			fileSystemSink, err := initFileSystemSink(mgr, config)
			if err != nil {
				return err
			}
			defer fileSystemSink.Stop()
			auditSinks = append(auditSinks, fileSystemSink)
			admissionSinks = append(admissionSinks, fileSystemSink)
		}

		if config.FluxNotificationSinkAddr != "" {
			logger.Info("initializing flux notification sink ...", "endpoint", config.FluxNotificationSinkAddr)
			fluxNotificationSink, err := initFluxNotificationSink(mgr, config)
			if err != nil {
				return err
			}
			defer fluxNotificationSink.Stop()
			if config.EnableAudit {
				logger.Warn("ignoring flux notifications sink for audit validation")
			}
			admissionSinks = append(admissionSinks, fluxNotificationSink)
		}

		if config.EnableK8sEventSink {
			logger.Info("initializing kubernetes events sink ...")
			k8sEventSink, err := initK8sEventSink(mgr, config)
			if err != nil {
				return err
			}
			defer k8sEventSink.Stop()
			if config.EnableAudit {
				logger.Warn("ignoring kubernetes events sink for audit validation")
			}
			admissionSinks = append(admissionSinks, k8sEventSink)
		}

		if config.GatewaySinkURL != "" {
			logger.Info("initializing SaaS gateway sink...")
			gateway, err := initSaaSGateway(contextCli.Context, kubeClient, config)
			if err != nil {
				return err
			}
			if config.EnableAudit {
				gatewaySink, err := initSaaSSink(contextCli.Context, mgr, kubeClient, config, gateway, packet.PacketPolicyValidationAudit)
				if err != nil {
					return err
				}
				auditSinks = append(auditSinks, gatewaySink)
			}
			if config.EnableAdmission {
				gatewaySink, err := initSaaSSink(contextCli.Context, mgr, kubeClient, config, gateway, packet.PacketPolicyValidationAdmission)
				if err != nil {
					return err
				}
				admissionSinks = append(admissionSinks, gatewaySink)
			}
		}

		if config.EnableAudit {
			logger.Info("starting audit policies watcher")

			policiesSource, err := crd.NewPoliciesWatcher(contextCli.Context, mgr)
			if err != nil {
				return fmt.Errorf("failed to initialize CRD policies source: %w", err)
			}

			if config.AuditPolicySet != "" {
				policiesSource.SetPolicySet(config.AuditPolicySet)
			}

			validator := validation.NewOPAValidator(
				policiesSource,
				config.WriteCompliance,
				auditor.TypeAudit,
				config.AccountID,
				config.ClusterID,
				auditSinks...,
			)

			auditController := auditor.NewAuditController(validator, auditControllerInterval, entitiesSources...)
			mgr.Add(auditController)
			auditController.Audit(auditor.AuditEventTypeInitial, nil)
		}

		if config.EnableAdmission {
			logger.Info("starting admission policies watcher")

			policiesSource, err := crd.NewPoliciesWatcher(contextCli.Context, mgr)
			if err != nil {
				return fmt.Errorf("failed to initialize CRD policies source: %w", err)
			}

			if config.AdmissionPolicySet != "" {
				policiesSource.SetPolicySet(config.AdmissionPolicySet)
			}

			validator := validation.NewOPAValidator(
				policiesSource,
				config.WriteCompliance,
				admission.TypeAdmission,
				config.AccountID,
				config.ClusterID,
				admissionSinks...,
			)
			admissionServer := admission.NewAdmissionHandler(
				config.LogLevel,
				validator,
			)
			logger.Info("starting admission server...")
			err = admissionServer.Run(mgr)
			if err != nil {
				return fmt.Errorf("failed to start admission server: %w", err)
			}
		}

		err = mgr.Start(ctrl.SetupSignalHandler())
		if err != nil {
			return fmt.Errorf("failed to run agent: %w", err)
		}

		return nil
	}
	err := app.Run(os.Args)
	if err != nil {
		logger.Fatal(err)
	}
}

func initFileSystemSink(mgr manager.Manager, config Config) (*filesystem.FileSystemSink, error) {
	sink, err := filesystem.NewFileSystemSink(config.FileSystemSinkFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize filesystem sink: %w", err)
	}

	logger.Info("starting file system sink ...")
	mgr.Add(sink)

	return sink, nil
}

func initFluxNotificationSink(mgr manager.Manager, config Config) (*flux_notification.FluxNotificationSink, error) {
	recorder, err := events.NewRecorder(mgr, mgr.GetLogger(), config.FluxNotificationSinkAddr, eventReportingController)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize event recorder: %w", err)
	}

	sink, err := flux_notification.NewFluxNotificationSink(recorder, config.FluxNotificationSinkAddr, config.AccountID, config.ClusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize flux notification sink: %w", err)
	}

	logger.Info("starting flux notification sink ...")
	mgr.Add(sink)

	return sink, nil
}

func initK8sEventSink(mgr manager.Manager, config Config) (*k8s_event.K8sEventSink, error) {
	clientset, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes clientset: %w", err)
	}

	sink, err := k8s_event.NewK8sEventSink(clientset, config.AccountID, config.ClusterID, eventReportingController)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes event sink: %w", err)
	}

	logger.Info("starting kubernetes event sink ...")
	mgr.Add(sink)

	return sink, nil
}

func initSaaSSink(ctx context.Context, mgr manager.Manager, kubeClient *kube.KubeClient, config Config, gateway *gateway.Gateway, packetKind packet.PacketKind) (*saas.SaaSGatewaySink, error) {
	sink := saas.NewSaaSGatewaySink(
		gateway,
		packetKind,
		SaaSSinkBatchSize,
		SaaSSinkBatchExpiry,
	)
	logger.Info("starting SaaS gateway connection")
	go gateway.Start(ctx)
	active := gateway.WaitActive(ctx, 10*time.Second)
	if !active {
		return nil, errors.New("timeout while waiting for SaaS gateway connection")
	}
	logger.Info("starting Saas gateway sink ...")
	mgr.Add(sink)

	return sink, nil
}

func initSaaSGateway(ctx context.Context, kubeClient *kube.KubeClient, config Config) (*gateway.Gateway, error) {
	secret, err := base64.StdEncoding.DecodeString(config.GatewaySinkSecret)
	if err != nil {
		return nil, errors.New("secret not encoded in base64 format")
	}
	gatewayURL, err := url.Parse(config.GatewaySinkURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse gateway url: %w", err)
	}
	accountID, err := uuid.FromString(config.AccountID)
	if err != nil {
		return nil, errors.New("invalid uuid format for account id")
	}
	clusterID, err := uuid.FromString(config.ClusterID)
	if err != nil {
		return nil, errors.New("invalid uuid format for cluster id")
	}

	var agentPermissions string
	permissionsObj, err := kubeClient.GetAgentPermissions(ctx)
	if err != nil {
		agentPermissions = err.Error()
		logger.Warnw("Failed to get agent permissions", "error", err)
	}

	agentPermissionsBytes, err := json.Marshal(permissionsObj.Status.ResourceRules)
	if err != nil {
		parseErr := fmt.Errorf("error while parsing agent permissions: %w", err)
		agentPermissions = parseErr.Error()
		logger.Warnw("Failed to parse agent permissions", "error", err)
	}

	agentPermissions = string(agentPermissionsBytes)

	k8sServerVersion, err := kubeClient.GetServerVersion()
	if err != nil {
		k8sServerVersion = err.Error()
		logger.Warnw("failed to discover kubernetes server version", "error", err)
	}

	clusterProvider, err := kubeClient.GetClusterProvider(ctx)
	if err != nil {
		clusterProvider = err.Error()
		logger.Warnw("failed to get kubernetes cluster provider", "error", err)
	}

	gateway := gateway.NewGateway(
		*gatewayURL,
		accountID,
		clusterID,
		secret,
		k8sServerVersion,
		clusterProvider,
		agentPermissions,
		build,
	)
	return gateway, nil
}
