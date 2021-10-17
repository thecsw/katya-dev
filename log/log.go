package log

import (
	"os"
	"time"

	stdlog "log"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm/logger"
)

var (
	// log is a global logrus instance.
	log = logrus.New()

	// DBLogger is the logger instance for our DB to prevent scam logging
	// the default logger spits out too much onto the screen
	DBLogger = logger.New(
		stdlog.New(os.Stdout, "\r\n", stdlog.LstdFlags), // io writer
		logger.Config{
			// Slow SQL threshold
			SlowThreshold: time.Second,
			// Log level
			LogLevel: logger.Silent,
			// Ignore ErrRecordNotFound error for logger
			IgnoreRecordNotFoundError: true,
			// Disable color
			Colorful: true,
		},
	)
)

// Params is an alias for `map[string]interface{}`.
type Params map[string]interface{}

// Init initializes the global logrus instance.
func Init() {
	log.SetFormatter(&logrus.TextFormatter{
		ForceColors:     true,
		ForceQuote:      true,
		FullTimestamp:   true,
		TimestampFormat: time.RFC1123,
		DisableSorting:  false,
	})
	log.SetOutput(os.Stdout)
	Info("Logger is created")
}

// Format is an INFO log output with fields.
func Format(what string, fields Params) {
	log.WithFields(logrus.Fields(fields)).Infoln(what)
}

// Error is an ERROR log output with fields.
func Error(msg string, err error, fields Params) {
	if err == nil {
		return
	}
	log.WithError(err).WithFields(logrus.Fields(fields)).Errorln(msg)
}

// Info is an INFO log output.
func Info(what interface{}) {
	log.Infoln(what)
}
