package log

import (
	"github.com/MagalixTechnologies/core/logger"
	"github.com/go-logr/logr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ControllerLogSink struct {
	accountID string
	clusterID string
	baseLog   *zap.Logger
	sugarLog  logger.Logger
}

func NewControllerLog(accountID, clusterID string) logr.Logger {
	sink := ControllerLogSink{
		accountID: accountID,
		clusterID: clusterID,
	}
	return logr.New(&sink)
}

func (c *ControllerLogSink) Init(info logr.RuntimeInfo) {
	log := logger.NewZapLogger(logger.InfoLevel)
	sugarLog := log.WithOptions(zap.AddCallerSkip(info.CallDepth + 1)).Sugar()
	sugarLog = sugarLog.With("accountID", c.accountID, "clusterID", c.clusterID)
	c.sugarLog = sugarLog
	c.baseLog = log
}

func (c *ControllerLogSink) Enabled(level int) bool {
	return c.baseLog.Core().Enabled(zapcore.Level(level))
}

func (c *ControllerLogSink) Info(_ int, msg string, keysAndValues ...interface{}) {
	c.sugarLog.Infow(msg, keysAndValues...)
}

func (c *ControllerLogSink) Error(err error, msg string, keysAndValues ...interface{}) {
	c.sugarLog.Errorw(msg, "error", err, keysAndValues)
}

func (c *ControllerLogSink) WithValues(keysAndValues ...interface{}) logr.LogSink {
	return &ControllerLogSink{
		accountID: c.accountID,
		clusterID: c.clusterID,
		baseLog:   c.baseLog,
		sugarLog:  c.sugarLog.With(keysAndValues...),
	}
}

func (c *ControllerLogSink) WithName(name string) logr.LogSink {
	return &ControllerLogSink{
		accountID: c.accountID,
		clusterID: c.clusterID,
		baseLog:   c.baseLog,
		sugarLog:  c.sugarLog.Named(name),
	}
}
