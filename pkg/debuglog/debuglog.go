// Package debuglog creates a structured logger to a debug file.
package debuglog

import (
	"os"

	"github.com/rs/zerolog"
)

const logPath = "/tmp/mass.log"

var logger *zerolog.Logger

// Log returns a singleton structured logger that writes to the debug log file.
func Log() *zerolog.Logger {
	if logger == nil {
		logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)

		if err != nil {
			panic(err)
		}

		newLogger := zerolog.New(logFile).With().Timestamp().Caller().Logger()
		logger = &newLogger
	}

	return logger
}
