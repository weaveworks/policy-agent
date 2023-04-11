package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Level int8

type Logger = *zap.SugaredLogger

const (
	DebugLevel Level = iota - 1
	InfoLevel
	WarnLevel
	ErrorLevel
)

func New(level Level) Logger {
	logger := NewZapLogger(level)
	return logger.Sugar()
}

func NewZapLogger(level Level) *zap.Logger {
	core := zap.NewProductionConfig()
	core.Level = zap.NewAtomicLevelAt(getLevel(level))
	core.EncoderConfig.TimeKey = "timestamp"
	core.EncoderConfig.MessageKey = "message"
	core.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logger, _ := core.Build()
	return logger
}

var log Logger

func init() {
	core := zap.NewProductionConfig()
	core.EncoderConfig.TimeKey = "timestamp"
	core.EncoderConfig.MessageKey = "message"
	core.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	customLog, _ := core.Build()
	log = customLog.WithOptions(zap.AddCallerSkip(1)).Sugar()
}

// Config sets configurations for global logger
func Config(level Level) {
	customLog := NewZapLogger(level)
	log = customLog.WithOptions(zap.AddCallerSkip(1)).Sugar()
}

// ConfigWriterSync sets configurations for global logger
func ConfigWriterSync(level Level, w zapcore.WriteSyncer) {
	customLog := zap.New(zapcore.NewCore(zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}), zapcore.NewMultiWriteSyncer(os.Stdout, w), zap.NewAtomicLevelAt(getLevel(level))))

	log.Desugar().Core()

	log = customLog.WithOptions(zap.AddCaller(), zap.AddCallerSkip(1)).Sugar()
}

// WithGlobal adds a variadic number of fields to the logging context. It accepts a
// mix of strongly-typed Field objects and loosely-typed key-value pairs. When
// processing pairs, the first element of the pair is used as the field key
// and the second as the field value.
//
// For example,
//   logger.With(
//     "hello", "world",
//     "failure", errors.New("oh no"),
//     Stack(),
//     "count", 42,
//     "user", User{Name: "alice"},
//  )
//
// Note that the keys in key-value pairs should be strings.
// If you pass a non-string key panics a separate error is logged, but the key-value pair is skipped
// and execution continues. Passing an orphaned key triggers similar behavior
func WithGlobal(args ...interface{}) {
	log = log.With(args...)
}

// With adds a variadic number of fields to the logging context. It accepts a
// mix of strongly-typed Field objects and loosely-typed key-value pairs. When
// processing pairs, the first element of the pair is used as the field key
// and the second as the field value.
//
// For example,
//   logger.With(
//     "hello", "world",
//     "failure", errors.New("oh no"),
//     Stack(),
//     "count", 42,
//     "user", User{Name: "alice"},
//  )
//
// Note that the keys in key-value pairs should be strings.
// If you pass a non-string key panics a separate error is logged, but the key-value pair is skipped
// and execution continues. Passing an orphaned key triggers similar behavior
func With(args ...interface{}) Logger {
	return log.With(args...)
}

// Sync flushes any buffered log entries.
func Sync() error {
	return log.Sync()
}

// Debug uses fmt.Sprint to construct and log a message.
func Debug(args ...interface{}) {
	log.Debug(args...)
}

// Info uses fmt.Sprint to construct and log a message.
func Info(args ...interface{}) {
	log.Info(args...)
}

// Print uses fmt.Sprint to construct and log a message.
func Print(args ...interface{}) {
	log.Info(args...)
}

// Println uses fmt.Sprint to construct and log a message.
func Println(args ...interface{}) {
	log.Info(args...)
}

// Warn uses fmt.Sprint to construct and log a message.
func Warn(args ...interface{}) {
	log.Warn(args...)
}

// Error uses fmt.Sprint to construct and log a message.
func Error(args ...interface{}) {
	log.Error(args...)
}

// DPanic uses fmt.Sprint to construct and log a message. In development, the
// logger then panics. (See DPanicLevel for details.)
func DPanic(args ...interface{}) {
	log.DPanic(args...)
}

// Panic uses fmt.Sprint to construct and log a message, then panics.
func Panic(args ...interface{}) {
	log.Panic(args...)
}

// Fatal uses fmt.Sprint to construct and log a message, then calls os.Exit.
func Fatal(args ...interface{}) {
	log.Fatal(args...)
}

// Debugf uses fmt.Sprintf to log a templated message.
func Debugf(template string, args ...interface{}) {
	log.Debugf(template, args...)
}

// Infof uses fmt.Sprintf to log a templated message.
func Infof(template string, args ...interface{}) {
	log.Infof(template, args...)
}

// Infof uses fmt.Sprintf to log a templated message.
func Printf(template string, args ...interface{}) {
	log.Infof(template, args...)
}

// Warnf uses fmt.Sprintf to log a templated message.
func Warnf(template string, args ...interface{}) {
	log.Warnf(template, args...)
}

// Errorf uses fmt.Sprintf to log a templated message.
func Errorf(template string, args ...interface{}) {
	log.Errorf(template, args...)
}

// DPanicf uses fmt.Sprintf to log a templated message. In development, the
// logger then panics. (See DPanicLevel for details.)
func DPanicf(template string, args ...interface{}) {
	log.DPanicf(template, args...)
}

// Panicf uses fmt.Sprintf to log a templated message, then panics.
func Panicf(template string, args ...interface{}) {
	log.Panicf(template, args...)
}

// Fatalf uses fmt.Sprintf to log a templated message, then calls os.Exit.
func Fatalf(template string, args ...interface{}) {
	log.Fatalf(template, args...)
}

// Debugw logs a message with some additional context. The variadic key-value
// pairs are treated as they are in With.
//
// When debug-level logging is disabled, this is much faster than
//  s.With(keysAndValues).Debug(msg)
func Debugw(msg string, keysAndValues ...interface{}) {
	log.Debugw(msg, keysAndValues...)
}

// Infow logs a message with some additional context. The variadic key-value
// pairs are treated as they are in With.
func Infow(msg string, keysAndValues ...interface{}) {
	log.Infow(msg, keysAndValues...)
}

// Warnw logs a message with some additional context. The variadic key-value
// pairs are treated as they are in With.
func Warnw(msg string, keysAndValues ...interface{}) {
	log.Warnw(msg, keysAndValues...)
}

// Errorw logs a message with some additional context. The variadic key-value
// pairs are treated as they are in With.
func Errorw(msg string, keysAndValues ...interface{}) {
	log.Errorw(msg, keysAndValues...)
}

// DPanicw logs a message with some additional context. In development, the
// logger then panics. (See DPanicLevel for details.) The variadic key-value
// pairs are treated as they are in With.
func DPanicw(msg string, keysAndValues ...interface{}) {
	log.DPanicw(msg, keysAndValues...)
}

// Panicw logs a message with some additional context, then panics. The
// variadic key-value pairs are treated as they are in With.
func Panicw(msg string, keysAndValues ...interface{}) {
	log.Panicw(msg, keysAndValues...)
}

// Fatalw logs a message with some additional context, then calls os.Exit. The
// variadic key-value pairs are treated as they are in With.
func Fatalw(msg string, keysAndValues ...interface{}) {
	log.Fatalw(msg, keysAndValues...)
}

func getLevel(level Level) zapcore.Level {
	switch level {
	case InfoLevel:
		return zapcore.InfoLevel
	case DebugLevel:
		return zapcore.DebugLevel
	case WarnLevel:
		return zapcore.WarnLevel
	case ErrorLevel:
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}
