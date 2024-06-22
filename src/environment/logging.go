package environment

import (
	"github.com/sirupsen/logrus"
)

func ConfigureLogger() {
	enableInternalLogger := GetEnvironmentVariable("DISPATCH_DEBUG_MODE")
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})

	loggingLevel := logrus.PanicLevel
	if enableInternalLogger == "true" {
		loggingLevel = logrus.DebugLevel
	}
	logrus.SetLevel(loggingLevel)
}

func GetInternalLogger() *logrus.Logger {
	return logrus.StandardLogger()
}
