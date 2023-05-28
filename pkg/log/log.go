package log

import (
	"github.com/go-logr/logr"
	"github.com/weaveworks/weave-policy-agent/pkg/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ControllerLogSink provides logging for the controller manager, implements github.com/go-logr/logr.LogSink
type ControllerLogSink struct {
	accountID string
	clusterID string
	baseLog   *zap.Logger
	sugarLog  logger.Logger
}

// NewControllerLog returns a logger for controller manager
func NewControllerLog(accountID, clusterID string) logr.Logger {
	sink := ControllerLogSink{
		accountID: accountID,
		clusterID: clusterID,
	}
	return logr.New(&sink)
}

// Init initializes the logger with the needed configuration
func (c *ControllerLogSink) Init(info logr.RuntimeInfo) {
	log := logger.NewZapLogger(logger.InfoLevel)
	sugarLog := log.WithOptions(zap.AddCallerSkip(info.CallDepth + 1)).Sugar()
	sugarLog = sugarLog.With("accountID", c.accountID, "clusterID", c.clusterID)
	c.sugarLog = sugarLog
	c.baseLog = log
}

// Enabled check if a log level is enabled
func (c *ControllerLogSink) Enabled(level int) bool {
	return c.baseLog.Core().Enabled(zapcore.Level(level))
}

// Info logs a non-error message with the given key/value pairs as context
func (c *ControllerLogSink) Info(_ int, msg string, keysAndValues ...interface{}) {
	c.sugarLog.Infow(msg, keysAndValues...)
}

// Error logs an error, with the given message and key/value pairs as context
func (c *ControllerLogSink) Error(err error, msg string, keysAndValues ...interface{}) {
	c.sugarLog.Errorw(msg, "error", err, keysAndValues)
}

// WithValues returns a new LogSink with additional key/value pairs
func (c *ControllerLogSink) WithValues(keysAndValues ...interface{}) logr.LogSink {
	return &ControllerLogSink{
		accountID: c.accountID,
		clusterID: c.clusterID,
		baseLog:   c.baseLog,
		sugarLog:  c.sugarLog.With(keysAndValues...),
	}
}

// WithName returns a new LogSink with the specified name appended
func (c *ControllerLogSink) WithName(name string) logr.LogSink {
	return &ControllerLogSink{
		accountID: c.accountID,
		clusterID: c.clusterID,
		baseLog:   c.baseLog,
		sugarLog:  c.sugarLog.Named(name),
	}
}
