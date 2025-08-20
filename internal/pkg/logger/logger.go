package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

type Logger struct {
	*logrus.Logger
}

func New(level, format string) *Logger {
	logger := logrus.New()

	// Set output
	logger.SetOutput(os.Stdout)

	// Set log level
	switch level {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "warn":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}

	// Set formatter
	if format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
			ForceColors:   true,
		})
	}

	return &Logger{Logger: logger}
}

func (l *Logger) Error(msg string, keysAndValues ...interface{}) {
	l.WithFields(parseFields(keysAndValues...)).Error(msg)
}

func (l *Logger) Warn(msg string, keysAndValues ...interface{}) {
	l.WithFields(parseFields(keysAndValues...)).Warn(msg)
}

func (l *Logger) Info(msg string, keysAndValues ...interface{}) {
	l.WithFields(parseFields(keysAndValues...)).Info(msg)
}

func (l *Logger) Debug(msg string, keysAndValues ...interface{}) {
	l.WithFields(parseFields(keysAndValues...)).Debug(msg)
}

func parseFields(keysAndValues ...interface{}) logrus.Fields {
	fields := logrus.Fields{}
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			if key, ok := keysAndValues[i].(string); ok {
				fields[key] = keysAndValues[i+1]
			}
		}
	}
	return fields
}
