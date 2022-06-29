package configuration

import (
	"path/filepath"
	"strings"

	"github.com/MagalixTechnologies/core/logger"
	"github.com/spf13/viper"
)

type SinksConfig struct {
	FilesystemSink       *FileSystemSink
	FluxNotificationSink *FluxNotificationSink
	K8sEventSink         *K8sEventsSink
	SaasGatewaySink      *SaaSGatewaySink
}

type SaaSGatewaySink struct {
	URL    string
	Secret string
}

type K8sEventsSink struct {
	Enabled bool
}

type FileSystemSink struct {
	FilePath string
}

type FluxNotificationSink struct {
	Address string
}

type AdmissionWebhook struct {
	Listen  int
	CertDir string
}

type AdmissionConfig struct {
	Enabled   bool
	Webhook   AdmissionWebhook
	Sinks     SinksConfig
	PolicySet string
}

type AuditConfig struct {
	WriteCompliance bool
	Enabled         bool
	Sinks           SinksConfig
	PolicySet       string
}

type Config struct {
	KubeConfigFile string
	AccountID      string
	ClusterID      string

	LogLevel string

	ProbesListen   string
	MetricsAddress string

	Admission AdmissionConfig
	Audit     AuditConfig
}

func GetAgentConfiguration(filePath string) Config {
	dir, file := filepath.Split(filePath)

	viper.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)

	viper.SetConfigName(file)
	viper.SetConfigType("toml")
	viper.AddConfigPath(dir)

	err := viper.ReadInConfig()
	if err != nil {
		logger.Fatal(err)
	}
	viper.SetDefault("kubeConfigFile", "")
	viper.SetDefault("metricsAddress", ":8080")
	viper.SetDefault("probesListen", ":9000")
	viper.SetDefault("logLevel", "info")
	viper.SetDefault("admission.webhook.listen", 8443)
	viper.SetDefault("admission.webhook.certDir", "/certs")

	checkRequiredFields()

	c := Config{}

	err = viper.Unmarshal(&c)
	if err != nil {
		logger.Fatal(err)
	}

	if c.Admission.Enabled && c.Admission.Sinks.SaasGatewaySink != nil {
		c.Admission.Sinks.SaasGatewaySink.URL = viper.Get(
			"admission.sinks.saasGatewaySink.url").(string)
		c.Admission.Sinks.SaasGatewaySink.Secret = viper.Get(
			"admission.sinks.saasGatewaySink.secret").(string)
	}

	if c.Audit.Enabled && c.Audit.Sinks.SaasGatewaySink != nil {
		c.Audit.Sinks.SaasGatewaySink.URL = viper.Get(
			"audit.sinks.saasGatewaySink.url").(string)
		c.Audit.Sinks.SaasGatewaySink.Secret = viper.Get(
			"audit.sinks.saasGatewaySink.secret").(string)
	}

	return c
}

func checkRequiredFields() {
	requiredFields := []string{
		"accountId",
		"clusterId",
	}

	for _, field := range requiredFields {
		if !viper.IsSet(field) {
			logger.Fatalw(
				"missing key in agent configuration file",
				"key", field,
			)
		}
	}
}
