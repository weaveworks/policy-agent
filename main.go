package main

import (
	"os"

	"github.com/MagalixCorp/new-magalix-agent/admission"
	magalixv1 "github.com/MagalixCorp/new-magalix-agent/apiextensions/magalix.com/v1"
	policiesClient "github.com/MagalixCorp/new-magalix-agent/clients/magalix.com/v1"
	"github.com/MagalixCorp/new-magalix-agent/policies/crd"
	"github.com/MagalixCorp/new-magalix-agent/sink/logging"
	"github.com/MagalixTechnologies/core/logger"
	"github.com/urfave/cli/v2"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Config struct {
	KubeConfigFile  string
	WriteCompliance bool
	WebhookListen   string
	WebhookCertFile string
	WebhhokKeyFile  string
}

// @TODO retrieve account and cluster ids and add them in result?
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
		},
		&cli.StringFlag{
			Name:        "webhook-listen",
			Usage:       "address for the admission webhook server to listen on",
			Destination: &config.WebhookListen,
			Value:       ":8443",
		},
		&cli.StringFlag{
			Name:        "webhook-cert-file",
			Usage:       "cert file path for webhook server",
			Destination: &config.WebhookCertFile,
			Value:       "tls.crt",
		},
		&cli.StringFlag{
			Name:        "webhook-key-file",
			Usage:       "key file path for webhook server",
			Destination: &config.WebhhokKeyFile,
			Value:       "tls.key",
		},
		&cli.BoolFlag{
			Name:        "write-compliance",
			Usage:       "enables writing compliance results",
			Destination: &config.WriteCompliance,
			Value:       false,
		},
	}

	app.Action = func(contextCli *cli.Context) error {
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

		kubePoliciesClient := policiesClient.NewKubePoliciesClient(kubeConfig)

		policiesSource, err := crd.NewPoliciesCRD(kubePoliciesClient)
		if err != nil {
			logger.Fatalw("failed to initialize CRD policies source", "error", err)
		}
		defer policiesSource.Close()

		logSink := logging.NewLogSink()

		admissionServer := admission.NewAdmissionHandler(
			config.WebhookListen,
			config.WebhookCertFile,
			config.WebhhokKeyFile,
			policiesSource,
			config.WriteCompliance,
			logSink,
		)
		logger.Info("Starting admission server...")
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
