package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	magalixv1 "github.com/MagalixCorp/magalix-policy-agent/apiextensions/magalix.com/v1"
	"github.com/MagalixCorp/magalix-policy-agent/auditor"
	"github.com/MagalixCorp/magalix-policy-agent/clients/kube"
	policiesClient "github.com/MagalixCorp/magalix-policy-agent/clients/magalix.com/v1"
	"github.com/MagalixCorp/magalix-policy-agent/entities/k8s"
	"github.com/MagalixCorp/magalix-policy-agent/pkg/domain"
	"github.com/MagalixCorp/magalix-policy-agent/pkg/validation"
	"github.com/MagalixCorp/magalix-policy-agent/policies/crd"
	"github.com/MagalixCorp/magalix-policy-agent/server/admission"
	"github.com/MagalixCorp/magalix-policy-agent/server/probes"
	"github.com/MagalixCorp/magalix-policy-agent/sink/filesystem"
	flux_notification "github.com/MagalixCorp/magalix-policy-agent/sink/flux-notification"
	k8s_event "github.com/MagalixCorp/magalix-policy-agent/sink/k8s-event"
	"github.com/MagalixTechnologies/core/logger"
	"github.com/fluxcd/pkg/runtime/events"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
	"k8s.io/client-go/kubernetes"
	scheme_client "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Config struct {
	KubeConfigFile   string
	AccountID        string
	ClusterID        string
	WriteCompliance  bool
	WebhookListen    string
	WebhookCertFile  string
	WebhookKeyFile   string
	LogLevel         string
	ProbesListen     string
	DisableAdmission bool
	DisableAudit     bool

	// filesystem sink config
	EnableFileSystemSink   bool
	FileSystemSinkFilePath string

	// kubernets event sink config
	EnableK8sEventSink bool

	// flux notification sink config
	EnableFluxNotificationSink bool
	FluxNotificationSinkAddr   string
}

const (
	auditControllerInterval         = 23 * time.Hour
	eventReportingController string = "magalix-policy-agent"
)

func main() {
	config := Config{}
	app := cli.NewApp()
	app.Version = "0.0.1"
	app.Name = "Magalix agent"
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
		&cli.StringFlag{
			Name:        "webhook-listen",
			Usage:       "address for the admission webhook server to listen on",
			Destination: &config.WebhookListen,
			Value:       ":8443",
			EnvVars:     []string{"AGENT_WEBHOOK_LISTEN"},
		},
		&cli.StringFlag{
			Name:        "webhook-cert-file",
			Usage:       "cert file path for webhook server",
			Destination: &config.WebhookCertFile,
			Value:       "/certs/tls.crt",
			EnvVars:     []string{"AGENT_WEBHOOK_CERT_FILE"},
		},
		&cli.StringFlag{
			Name:        "webhook-key-file",
			Usage:       "key file path for webhook server",
			Destination: &config.WebhookKeyFile,
			Value:       "/certs/tls.key",
			EnvVars:     []string{"AGENT_WEBHOOK_KEY_FILE"},
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
		&cli.BoolFlag{
			Name:        "enable-filesystem-sink",
			Usage:       "enables filesystem sink",
			Destination: &config.EnableFileSystemSink,
			Value:       false,
			EnvVars:     []string{"AGENT_ENABLE_FILESYSTEM_SINK"},
		},
		&cli.StringFlag{
			Name:        "filesystem-sink-file-path",
			Usage:       "filesystem sink file path",
			Value:       "/tmp/results.json",
			Destination: &config.FileSystemSinkFilePath,
			EnvVars:     []string{"AGENT_FILESYSTEM_SINK_FILE_PATH"},
		},
		&cli.BoolFlag{
			Name:        "enable-flux-notification-sink",
			Usage:       "enables flux notification sink",
			Destination: &config.EnableFluxNotificationSink,
			Value:       false,
			EnvVars:     []string{"AGENT_ENABLE_FLUX_NOTIFICATION_SINK"},
		},
		&cli.StringFlag{
			Name:        "flux-notification-sink-addr",
			Usage:       "flux notification sink address",
			Value:       "http://notification-controller.flux-system.svc.cluster.local",
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

	app.Action = func(cli *cli.Context) error {
		logger.Info("initializing Magalix Policy Agent")
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

		scheme := scheme_client.Scheme
		err = magalixv1.AddToScheme(scheme)
		if err != nil {
			return fmt.Errorf("failed to add policy crd to schema: %w", err)
		}

		mgr, err := ctrl.NewManager(kubeConfig, ctrl.Options{
			Scheme:             scheme,
			MetricsBindAddress: "0",
		})

		probeHandler := probes.NewProbesHandler(config.ProbesListen)
		go func() {
			err := probeHandler.Run(cli.Context)
			if err != nil {
				logger.Fatal("failed to start probes server", "error", err)
			}
		}()

		kubePoliciesClient := policiesClient.NewKubePoliciesClient(kubeConfig)
		kubeClient, err := kube.NewKubeClient(kubeConfig)
		if err != nil {
			return fmt.Errorf("init client failed: %w", err)
		}
		entitiesSources, err := k8s.GetEntitiesSources(cli.Context, kubeClient)
		if err != nil {
			return fmt.Errorf("initializing entities sources failed: %w", err)
		}

		logger.Info("starting policies CRD watcher")
		policiesSource, err := crd.NewPoliciesWatcher(kubePoliciesClient)
		if err != nil {
			return fmt.Errorf("failed to initialize CRD policies source: %w", err)
		}
		defer policiesSource.Close()

		sinks := []domain.PolicyValidationSink{}

		if config.EnableFileSystemSink {
			logger.Info("initializing filesystem sink ...")
			fileSystemSink, err := initFileSystemSink(cli.Context, config)
			if err != nil {
				return err
			}
			defer fileSystemSink.Stop()
			sinks = append(sinks, fileSystemSink)
		}

		if config.EnableFluxNotificationSink {
			logger.Info("initializing flux notification sink ...")
			fkuxNotificationSink, err := initFluxNotificationSink(cli.Context, config, mgr)
			if err != nil {
				return err
			}
			defer fkuxNotificationSink.Stop()
			sinks = append(sinks, fkuxNotificationSink)
		}

		if config.EnableK8sEventSink {
			logger.Info("initializing kubernetes events sink ...")
			k8sEventSink, err := initK8sEventSink(cli.Context, config, kubeConfig)
			if err != nil {
				return err
			}
			defer k8sEventSink.Stop()
			sinks = append(sinks, k8sEventSink)
		}

		probeHandler.MarkReady(true)
		eg, ctx := errgroup.WithContext(cli.Context)

		if !config.DisableAudit {
			validator := validation.NewOPAValidator(
				policiesSource,
				config.WriteCompliance,
				auditor.TypeAudit,
				sinks...,
			)
			auditController := auditor.NewAuditController(validator, auditControllerInterval, entitiesSources...)
			eg.Go(func() error {
				logger.Info("starting audit controller...")
				return auditController.Run(ctx)
			})
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
				config.WebhookListen,
				config.WebhookCertFile,
				config.WebhookKeyFile,
				config.LogLevel,
				validator,
			)
			eg.Go(func() error {
				logger.Info("starting admission server...")
				err := admissionServer.Run(ctx)
				if err != nil {
					return fmt.Errorf("failed to start admission server: %w", err)
				}
				return nil
			})
		}

		err = eg.Wait()
		if err != nil {
			return err
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
