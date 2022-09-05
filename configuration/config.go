package configuration

import (
	"path/filepath"
	"strings"

	"github.com/MagalixTechnologies/core/logger"
	"github.com/spf13/viper"
)

type ValidationSinksConfig struct {
	FilesystemSink       *FileSystemSink
	FluxNotificationSink *FluxNotificationSink
	K8sEventsSink        *K8sEventsSink
	SaasGatewaySink      *SaaSGatewaySink
	ElasticSink          *ElasticSink
}

type SaaSGatewaySink struct {
	URL    string
	Secret string
}

type K8sEventsSink struct {
	Enabled bool
}

type FileSystemSink struct {
	FileName string
}

type FluxNotificationSink struct {
	Address string
}

type AdmissionWebhook struct {
	Listen  int
	CertDir string
}

type ElasticSink struct {
	IndexName     string
	Address       string
	Username      string
	Password      string
	InsertionMode string
}

type AdmissionModeConfig struct {
	Enabled         bool
	Webhook         AdmissionWebhook
	ValidationSinks ValidationSinksConfig
	PolicySet       string
}

type AuditModeConfig struct {
	WriteCompliance bool
	Enabled         bool
	ValidationSinks ValidationSinksConfig
	PolicySet       string
}

type Config struct {
	KubeConfigFile string
	AccountID      string
	ClusterID      string

	LogLevel string

	ProbesListen   string
	MetricsAddress string

	AdmissionMode AdmissionModeConfig
	AuditMode     AuditModeConfig
}

func GetAgentConfiguration(filePath string) Config {
	dir, file := filepath.Split(filePath)

	viper.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)

	viper.SetConfigName(file)
	viper.SetConfigType("yaml")
	viper.AddConfigPath(dir)

	err := viper.ReadInConfig()
	if err != nil {
		logger.Fatal(err)
	}
	viper.SetDefault("kubeConfigFile", "")
	viper.SetDefault("metricsAddress", ":8080")
	viper.SetDefault("probesListen", ":9000")
	viper.SetDefault("logLevel", "info")
	viper.SetDefault("admissionMode.webhook.listen", 8443)
	viper.SetDefault("admissionMode.webhook.certDir", "/certs")

	checkRequiredFields()

	c := Config{}

	err = viper.Unmarshal(&c)
	if err != nil {
		logger.Fatal(err)
	}

	if c.AdmissionMode.Enabled && c.AdmissionMode.ValidationSinks.SaasGatewaySink != nil {
		c.AdmissionMode.ValidationSinks.SaasGatewaySink.URL = getField(
			"admissionMode.validationSinks.saasGatewaySink.url")
		c.AdmissionMode.ValidationSinks.SaasGatewaySink.Secret = getField(
			"admissionMode.validationSinks.saasGatewaySink.secret")
	}

	if c.AuditMode.Enabled && c.AuditMode.ValidationSinks.SaasGatewaySink != nil {
		c.AuditMode.ValidationSinks.SaasGatewaySink.URL = getField(
			"auditMode.validationSinks.saasGatewaySink.url")
		c.AuditMode.ValidationSinks.SaasGatewaySink.Secret = getField(
			"auditMode.validationSinks.saasGatewaySink.secret")
	}

	if c.AdmissionMode.Enabled && c.AdmissionMode.ValidationSinks.ElasticSink != nil {
		c.AdmissionMode.ValidationSinks.ElasticSink.Address = getField(
			"admissionMode.validationSinks.elasticSink.address")
		c.AdmissionMode.ValidationSinks.ElasticSink.IndexName = getField(
			"admissionMode.validationSinks.elasticSink.indexname")
		c.AdmissionMode.ValidationSinks.ElasticSink.Username = getField(
			"admissionMode.validationSinks.elasticSink.username")
		c.AdmissionMode.ValidationSinks.ElasticSink.Password = getField(
			"admissionMode.validationSinks.elasticSink.password")
	}

	if c.AuditMode.Enabled && c.AuditMode.ValidationSinks.ElasticSink != nil {
		c.AuditMode.ValidationSinks.ElasticSink.Address = getField(
			"auditMode.validationSinks.elasticSink.address")
		c.AuditMode.ValidationSinks.ElasticSink.IndexName = getField(
			"auditMode.validationSinks.elasticSink.indexname")
		c.AuditMode.ValidationSinks.ElasticSink.Username = getField(
			"auditMode.validationSinks.elasticSink.username")
		c.AuditMode.ValidationSinks.ElasticSink.Password = getField(
			"auditMode.validationSinks.elasticSink.password")
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

func getField(key string) string {
	value, _ := viper.Get(key).(string)
	return value
}
