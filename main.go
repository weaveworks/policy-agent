package main

import (
	"fmt"
	"os"

	magalixv1 "github.com/MagalixCorp/magalix-policy-agent/apiextensions/magalix.com/v1"
	policiesClient "github.com/MagalixCorp/magalix-policy-agent/clients/magalix.com/v1"
	"github.com/MagalixCorp/magalix-policy-agent/pkg/validation"
	"github.com/MagalixCorp/magalix-policy-agent/policies/crd"
	"github.com/MagalixCorp/magalix-policy-agent/server/admission"
	"github.com/MagalixCorp/magalix-policy-agent/server/probes"
	"github.com/MagalixCorp/magalix-policy-agent/sink/filesystem"
	"github.com/MagalixTechnologies/core/logger"
	"github.com/urfave/cli/v2"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Config struct {
	KubeConfigFile  string
	AccountID       string
	ClusterID       string
	WriteCompliance bool
	WebhookListen   string
	WebhookCertFile string
	WebhhokKeyFile  string
	LogLevel        string
	SinkFilePath    string
	ProbesListen    string
}

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
			Destination: &config.WebhhokKeyFile,
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
		&cli.StringFlag{
			Name:        "log-level",
			Usage:       "app log level",
			Destination: &config.LogLevel,
			Value:       "info",
			EnvVars:     []string{"AGENT_LOG_LEVEL"},
		},
		&cli.StringFlag{
			Name:        "sink-file-path",
			Usage:       "file path to write validation result to",
			Destination: &config.SinkFilePath,
			Value:       "/var/results.json", //@TODO remove default value and only add sink when a value is specified
			EnvVars:     []string{"AGENT_SINK_FILE_PATH"},
		},
	}

	app.Before = func(c *cli.Context) error {
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
			logger.Fatalw("failed to load Kubernetes config", "error", err)
		}

		magalixv1.AddToScheme(scheme.Scheme)

		probeHandler := probes.NewProbesHandler(config.ProbesListen)
		go func() {
			err := probeHandler.Run(contextCli.Context)
			if err != nil {
				logger.Fatal("Failed to start probes server")
			}
		}()

		kubePoliciesClient := policiesClient.NewKubePoliciesClient(kubeConfig)

		logger.Info("starting policies CRD watcher")
		policiesSource, err := crd.NewPoliciesCRD(kubePoliciesClient)
		if err != nil {
			logger.Fatalw("failed to initialize CRD policies source", "error", err)
		}
		defer policiesSource.Close()

		fileSystemSink, err := filesystem.NewFileSystemSink(config.SinkFilePath, config.AccountID, config.ClusterID)
		if err != nil {
			logger.Fatalw("failed to initialize file system sink", "error", err)
		}

		logger.Info("starting file system sink")
		err = fileSystemSink.Start(contextCli.Context)
		if err != nil {
			logger.Fatalw("failed to start file system sink", "error", err)
		}
		defer fileSystemSink.Stop()

		validator := validation.NewOpaValidator(
			policiesSource,
			config.WriteCompliance,
			fileSystemSink,
		)

		admissionServer := admission.NewAdmissionHandler(
			config.WebhookListen,
			config.WebhookCertFile,
			config.WebhhokKeyFile,
			config.LogLevel,
			validator,
		)

		probeHandler.SetReady()
		logger.Info("starting admission server...")
		err = admissionServer.Run(contextCli.Context)
		if err != nil {
			logger.Fatalw("failed to start admission server", "error", err)
		}
		return nil
	}
	err := app.Run(os.Args)
	if err != nil {
		logger.Fatal(err)
	}
}
