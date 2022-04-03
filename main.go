package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/MagalixTechnologies/core/logger"
	"github.com/MagalixTechnologies/policy-core/domain"
	"github.com/MagalixTechnologies/policy-core/validation"
	"github.com/fluxcd/pkg/runtime/events"
	"github.com/urfave/cli/v2"
	policiesv1 "github.com/weaveworks/policy-agent/api/v1"
	"github.com/weaveworks/policy-agent/internal/admission"
	"github.com/weaveworks/policy-agent/internal/auditor"
	"github.com/weaveworks/policy-agent/internal/clients/kube"
	"github.com/weaveworks/policy-agent/internal/entities/k8s"
	"github.com/weaveworks/policy-agent/internal/policies/crd"
	"github.com/weaveworks/policy-agent/internal/sink/filesystem"
	flux_notification "github.com/weaveworks/policy-agent/internal/sink/flux-notification"
	k8s_event "github.com/weaveworks/policy-agent/internal/sink/k8s-event"
	"github.com/weaveworks/policy-agent/pkg/log"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

type Config struct {
	KubeConfigFile   string
	AccountID        string
	ClusterID        string
	WriteCompliance  bool
	WebhookListen    int
	WebhookCertDir   string
	LogLevel         string
	ProbesListen     string
	DisableAdmission bool
	DisableAudit     bool

	// filesystem sink config
	FileSystemSinkFilePath string

	// kubernets event sink config
	EnableK8sEventSink bool

	// flux notification sink config
	FluxNotificationSinkAddr string
	MetricsAddr              string
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
			Name:        "disable-admission",
			Usage:       "disables admission control",
			Destination: &config.DisableAdmission,
			Value:       false,
			EnvVars:     []string{"AGENT_DISABLE_ADMISSION"},
		},
		&cli.BoolFlag{
			Name:        "disable-audit",
			Usage:       "disables cluster periodical audit",
			Destination: &config.DisableAudit,
			Value:       false,
			EnvVars:     []string{"AGENT_DISABLE_AUDIT"},
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
			Name:        "metrics-addr",
			Usage:       "address the metric endpoint binds to",
			Destination: &config.MetricsAddr,
			Value:       ":8080",
			EnvVars:     []string{"AGENT_METRICS_ADDR"},
		},
	}

	app.Before = func(c *cli.Context) error {
		if config.DisableAdmission && config.DisableAudit {
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
		logger.Info("initializing Policy Agent")
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

		err = policiesv1.AddToScheme(scheme)
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

		logger.Info("starting policies CRD watcher")
		policiesSource, err := crd.NewPoliciesWatcher(contextCli.Context, mgr)
		if err != nil {
			return fmt.Errorf("failed to initialize CRD policies source: %w", err)
		}

		sinks := []domain.PolicyValidationSink{}

		if config.FileSystemSinkFilePath != "" {
			logger.Infow("initializing filesystem sink ...", "file", config.FileSystemSinkFilePath)
			fileSystemSink, err := initFileSystemSink(contextCli.Context, config)
			if err != nil {
				return err
			}
			defer fileSystemSink.Stop()
			sinks = append(sinks, fileSystemSink)
		}

		if config.FluxNotificationSinkAddr != "" {
			logger.Info("initializing flux notification sink ...", "endpoint", config.FluxNotificationSinkAddr)
			fkuxNotificationSink, err := initFluxNotificationSink(contextCli.Context, config, mgr)
			if err != nil {
				return err
			}
			defer fkuxNotificationSink.Stop()
			sinks = append(sinks, fkuxNotificationSink)
		}

		if config.EnableK8sEventSink {
			logger.Info("initializing kubernetes events sink ...")
			k8sEventSink, err := initK8sEventSink(contextCli.Context, config, kubeConfig)
			if err != nil {
				return err
			}
			defer k8sEventSink.Stop()
			sinks = append(sinks, k8sEventSink)
		}

		if !config.DisableAudit {
			validator := validation.NewOPAValidator(
				policiesSource,
				config.WriteCompliance,
				auditor.TypeAudit,
				sinks...,
			)
			auditController := auditor.NewAuditController(validator, auditControllerInterval, entitiesSources...)
			mgr.Add(auditController)
			auditController.Audit(auditor.AuditEventTypeInitial, nil)
		}

		if !config.DisableAdmission {
			validator := validation.NewOPAValidator(
				policiesSource,
				config.WriteCompliance,
				admission.TypeAdmission,
				sinks...,
			)
			admissionServer := admission.NewAdmissionHandler(
				config.LogLevel,
				validator,
			)
			logger.Info("starting admission server...")
			err := admissionServer.Run(mgr)
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

func initFileSystemSink(ctx context.Context, config Config) (*filesystem.FileSystemSink, error) {
	sink, err := filesystem.NewFileSystemSink(config.FileSystemSinkFilePath, config.AccountID, config.ClusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize filesystem sink: %w", err)
	}

	logger.Info("starting file system sink ...")
	err = sink.Start(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start filesystem sink: %w", err)
	}

	return sink, nil
}

func initFluxNotificationSink(ctx context.Context, config Config, mgr ctrl.Manager) (*flux_notification.FluxNotificationSink, error) {
	recorder, err := events.NewRecorder(mgr, mgr.GetLogger(), config.FluxNotificationSinkAddr, eventReportingController)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize event recorder: %w", err)
	}

	sink, err := flux_notification.NewFluxNotificationSink(recorder, config.FluxNotificationSinkAddr, config.AccountID, config.ClusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize flux notification sink: %w", err)
	}

	logger.Info("starting flux notification sink ...")
	sink.Start(ctx)

	return sink, nil
}

func initK8sEventSink(ctx context.Context, config Config, kubeConfig *rest.Config) (*k8s_event.K8sEventSink, error) {
	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes clientset: %w", err)
	}

	sink, err := k8s_event.NewK8sEventSink(clientset, config.AccountID, config.ClusterID, eventReportingController)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes event sink: %w", err)
	}

	logger.Info("starting kubernetes event sink ...")
	sink.Start(ctx)

	return sink, nil
}
