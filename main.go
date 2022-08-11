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
	pacv2 "github.com/weaveworks/policy-agent/api/v2beta1"
	"github.com/weaveworks/policy-agent/configuration"
	"github.com/weaveworks/policy-agent/internal/admission"
	"github.com/weaveworks/policy-agent/internal/auditor"
	"github.com/weaveworks/policy-agent/internal/clients/gateway"
	"github.com/weaveworks/policy-agent/internal/clients/kube"
	"github.com/weaveworks/policy-agent/internal/entities/k8s"
	"github.com/weaveworks/policy-agent/internal/policies/crd"
	"github.com/weaveworks/policy-agent/internal/sink/elastic"
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

var (
	scheme = runtime.NewScheme()
)

const (
	auditControllerInterval         = 23 * time.Hour
	eventReportingController string = "policy-agent"
	configFilePath           string = "/config/config.yaml"
)

func main() {
	var config configuration.Config

	app := cli.NewApp()
	app.Version = "0.0.1"
	app.Name = "Policy agent"
	app.Usage = "Enforces compliance on your kubernetes cluster"

	app.Before = func(c *cli.Context) error {
		config = configuration.GetAgentConfiguration(configFilePath)

		if !config.Admission.Enabled && !config.Audit.Enabled {
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

		err = pacv2.AddToScheme(scheme)
		if err != nil {
			return fmt.Errorf("failed to add policy crd to scheme: %w", err)
		}

		lg := log.NewControllerLog(config.AccountID, config.ClusterID)

		mgr, err := ctrl.NewManager(kubeConfig, ctrl.Options{
			Scheme:                 scheme,
			MetricsBindAddress:     config.MetricsAddress,
			Port:                   config.Admission.Webhook.Listen,
			CertDir:                config.Admission.Webhook.CertDir,
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

		var auditSaaSGatewaySink, admissionSaaSGatewaySink *configuration.SaaSGatewaySink

		if config.Audit.Enabled {
			auditSinksConfig := config.Audit.Sinks
			if auditSinksConfig.FilesystemSink != nil {
				filePath := auditSinksConfig.FilesystemSink.FilePath
				logger.Infow("initializing filesystem audit sink ...", "file", filePath)
				fileSystemSink, err := initFileSystemSink(mgr, filePath)
				if err != nil {
					return err
				}
				defer fileSystemSink.Stop()
				auditSinks = append(auditSinks, fileSystemSink)
			}
			if auditSinksConfig.K8sEventsSink != nil && auditSinksConfig.K8sEventsSink.Enabled {
				logger.Info("initializing kubernetes events audit sink ...")
				k8sEventSink, err := initK8sEventSink(mgr, config)
				if err != nil {
					return err
				}
				defer k8sEventSink.Stop()
				auditSinks = append(auditSinks, k8sEventSink)
			}
			if auditSinksConfig.FluxNotificationSink != nil {
				fluxControllerAddress := auditSinksConfig.FluxNotificationSink.Address
				logger.Info("initializing flux notification controller audit sink ...", "address", fluxControllerAddress)
				fluxNotificationSink, err := initFluxNotificationSink(mgr, config, fluxControllerAddress)
				if err != nil {
					return err
				}
				defer fluxNotificationSink.Stop()
				auditSinks = append(auditSinks, fluxNotificationSink)
			}
			if auditSinksConfig.ElasticSink != nil {
				elasticsearchSinkConfig := auditSinksConfig.ElasticSink
				elasticsearchSink, err := initElasticSearchSink(mgr, *elasticsearchSinkConfig)
				if err != nil {
					return err
				}
				auditSinks = append(auditSinks, elasticsearchSink)
			}
			if auditSinksConfig.SaasGatewaySink != nil {
				auditSaaSGatewaySink = auditSinksConfig.SaasGatewaySink
			}
		}

		if config.Admission.Enabled {
			admissionSinksConfig := config.Admission.Sinks
			if admissionSinksConfig.FilesystemSink != nil {
				filePath := admissionSinksConfig.FilesystemSink.FilePath
				logger.Infow("initializing filesystem admission sink ...", "file", filePath)
				fileSystemSink, err := initFileSystemSink(mgr, filePath)
				if err != nil {
					return err
				}
				defer fileSystemSink.Stop()
				admissionSinks = append(admissionSinks, fileSystemSink)
			}
			if admissionSinksConfig.K8sEventsSink != nil && admissionSinksConfig.K8sEventsSink.Enabled {
				logger.Info("initializing kubernetes events admission sink ...")
				k8sEventSink, err := initK8sEventSink(mgr, config)
				if err != nil {
					return err
				}
				defer k8sEventSink.Stop()
				admissionSinks = append(admissionSinks, k8sEventSink)
			}
			if admissionSinksConfig.FluxNotificationSink != nil {
				fluxControllerAddress := admissionSinksConfig.FluxNotificationSink.Address
				logger.Info("initializing flux notification controller admission sink ...", "address", fluxControllerAddress)
				fluxNotificationSink, err := initFluxNotificationSink(mgr, config, fluxControllerAddress)
				if err != nil {
					return err
				}
				defer fluxNotificationSink.Stop()
				admissionSinks = append(admissionSinks, fluxNotificationSink)
			}
			if admissionSinksConfig.ElasticSink != nil {
				elasticsearchSinkConfig := admissionSinksConfig.ElasticSink
				elasticsearchSink, err := initElasticSearchSink(mgr, *elasticsearchSinkConfig)
				if err != nil {
					return err
				}
				auditSinks = append(auditSinks, elasticsearchSink)
			}

			if admissionSinksConfig.SaasGatewaySink != nil {
				admissionSaaSGatewaySink = admissionSinksConfig.SaasGatewaySink
			}
		}

		if auditSaaSGatewaySink != nil && admissionSaaSGatewaySink != nil &&
			auditSaaSGatewaySink.URL != admissionSaaSGatewaySink.URL {
			return errors.New("failed to initialize SaaS gateway sink: different saas gateway sink url in admission and audit sinks configuration")
		}

		var saasGatewaySink configuration.SaaSGatewaySink
		if auditSaaSGatewaySink != nil || admissionSaaSGatewaySink != nil {
			if auditSaaSGatewaySink != nil {
				saasGatewaySink = *auditSaaSGatewaySink
			} else if admissionSaaSGatewaySink != nil {
				saasGatewaySink = *admissionSaaSGatewaySink
			}

			logger.Info("initializing SaaS gateway sink...")
			gateway, err := initSaaSGateway(contextCli.Context, kubeClient, config, saasGatewaySink)
			if err != nil {
				return err
			}
			if auditSaaSGatewaySink != nil {
				gatewaySink, err := initSaaSSink(contextCli.Context, mgr, gateway, packet.PacketPolicyValidationAudit)
				if err != nil {
					return err
				}
				auditSinks = append(auditSinks, gatewaySink)
			}
			if admissionSaaSGatewaySink != nil {
				gatewaySink, err := initSaaSSink(contextCli.Context, mgr, gateway, packet.PacketPolicyValidationAdmission)
				if err != nil {
					return err
				}
				admissionSinks = append(admissionSinks, gatewaySink)
			}
		}

		if config.Audit.Enabled {
			logger.Info("starting audit policies watcher")

			policiesSource, err := crd.NewPoliciesWatcher(contextCli.Context, mgr)
			if err != nil {
				return fmt.Errorf("failed to initialize CRD policies source: %w", err)
			}

			if config.Audit.PolicySet != "" {
				policiesSource.SetPolicySet(config.Audit.PolicySet)
			}

			validator := validation.NewOPAValidator(
				policiesSource,
				config.Audit.WriteCompliance,
				auditor.TypeAudit,
				config.AccountID,
				config.ClusterID,
				auditSinks...,
			)

			auditController := auditor.NewAuditController(validator, auditControllerInterval, entitiesSources...)
			mgr.Add(auditController)
			auditController.Audit(auditor.AuditEventTypeInitial, nil)
		}

		if config.Admission.Enabled {
			logger.Info("starting admission policies watcher")

			policiesSource, err := crd.NewPoliciesWatcher(contextCli.Context, mgr)
			if err != nil {
				return fmt.Errorf("failed to initialize CRD policies source: %w", err)
			}

			if config.Admission.PolicySet != "" {
				policiesSource.SetPolicySet(config.Admission.PolicySet)
			}

			validator := validation.NewOPAValidator(
				policiesSource,
				false,
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

func initFileSystemSink(mgr manager.Manager, filePath string) (*filesystem.FileSystemSink, error) {
	sink, err := filesystem.NewFileSystemSink(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize filesystem sink: %w", err)
	}

	logger.Info("starting file system sink ...")
	mgr.Add(sink)

	return sink, nil
}

func initFluxNotificationSink(mgr manager.Manager, config configuration.Config, fluxNotificationAddr string) (*flux_notification.FluxNotificationSink, error) {
	recorder, err := events.NewRecorder(mgr, mgr.GetLogger(), fluxNotificationAddr, eventReportingController)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize event recorder: %w", err)
	}

	sink, err := flux_notification.NewFluxNotificationSink(recorder, fluxNotificationAddr, config.AccountID, config.ClusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize flux notification sink: %w", err)
	}

	logger.Info("starting flux notification sink ...")
	mgr.Add(sink)

	return sink, nil
}

func initK8sEventSink(mgr manager.Manager, config configuration.Config) (*k8s_event.K8sEventSink, error) {
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

func initElasticSearchSink(mgr manager.Manager, elasticsearchSinkConfig configuration.ElasticSink) (*elastic.ElasticSearchSink, error) {
	if elasticsearchSinkConfig.InsertionMode != "insert" && elasticsearchSinkConfig.InsertionMode != "upsert" {
		return nil, errors.New("failed to initialize elasticsearch sink, insertion mode should be one of two options: insert or upsert")
	}
	sink, err := elastic.NewElasticSearchSink(elasticsearchSinkConfig.Address, elasticsearchSinkConfig.Username, elasticsearchSinkConfig.Password, elasticsearchSinkConfig.IndexName, elasticsearchSinkConfig.InsertionMode)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize elasticsearch sink: %w", err)
	}

	logger.Info("starting elasticsearch sink ...")
	mgr.Add(sink)

	return sink, nil
}

func initSaaSSink(ctx context.Context, mgr manager.Manager, gateway *gateway.Gateway, packetKind packet.PacketKind) (*saas.SaaSGatewaySink, error) {
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

func initSaaSGateway(ctx context.Context, kubeClient *kube.KubeClient, config configuration.Config, gatewaySink configuration.SaaSGatewaySink) (*gateway.Gateway, error) {
	secret, err := base64.StdEncoding.DecodeString(gatewaySink.Secret)
	if err != nil {
		return nil, errors.New("secret not encoded in base64 format")
	}
	gatewayURL, err := url.Parse(gatewaySink.URL)
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
