package configuration

import (
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"github.com/weaveworks/policy-agent/pkg/logger"
)

type SinksConfig struct {
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

type AdmissionConfig struct {
	Enabled bool
	Webhook AdmissionWebhook
	Sinks   SinksConfig
	Mutate  bool
}

type AuditConfig struct {
	WriteCompliance bool
	Enabled         bool
	Sinks           SinksConfig
	Interval        uint
}

type TFAdmissionConfig struct {
	Enabled bool
	Sinks   SinksConfig
}

type Config struct {
	KubeConfigFile string
	AccountID      string
	ClusterID      string

	LogLevel string

	ProbesListen   string
	MetricsAddress string

	Admission   AdmissionConfig
	Audit       AuditConfig
	TFAdmission TFAdmissionConfig
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
	viper.SetDefault("admission.webhook.listen", 8443)
	viper.SetDefault("admission.webhook.certDir", "/certs")
	viper.SetDefault("audit.interval", 24)

	checkRequiredFields()

	c := Config{}

	err = viper.Unmarshal(&c)
	if err != nil {
		logger.Fatal(err)
	}

	if c.Admission.Enabled && c.Admission.Sinks.SaasGatewaySink != nil {
		c.Admission.Sinks.SaasGatewaySink.URL = getField(
			"admission.sinks.saasGatewaySink.url")
		c.Admission.Sinks.SaasGatewaySink.Secret = getField(
			"admission.sinks.saasGatewaySink.secret")
	}

	if c.Audit.Enabled && c.Audit.Sinks.SaasGatewaySink != nil {
		c.Audit.Sinks.SaasGatewaySink.URL = getField(
			"audit.sinks.saasGatewaySink.url")
		c.Audit.Sinks.SaasGatewaySink.Secret = getField(
			"audit.sinks.saasGatewaySink.secret")
	}

	if c.Admission.Enabled && c.Admission.Sinks.ElasticSink != nil {
		c.Admission.Sinks.ElasticSink.Address = getField(
			"admission.sinks.elasticSink.address")
		c.Admission.Sinks.ElasticSink.IndexName = getField(
			"admission.sinks.elasticSink.indexname")
		c.Admission.Sinks.ElasticSink.Username = getField(
			"admission.sinks.elasticSink.username")
		c.Admission.Sinks.ElasticSink.Password = getField(
			"admission.sinks.elasticSink.password")
	}

	if c.Audit.Enabled && c.Audit.Sinks.ElasticSink != nil {
		c.Audit.Sinks.ElasticSink.Address = getField(
			"audit.sinks.elasticSink.address")
		c.Audit.Sinks.ElasticSink.IndexName = getField(
			"audit.sinks.elasticSink.indexname")
		c.Audit.Sinks.ElasticSink.Username = getField(
			"audit.sinks.elasticSink.username")
		c.Audit.Sinks.ElasticSink.Password = getField(
			"audit.sinks.elasticSink.password")
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
