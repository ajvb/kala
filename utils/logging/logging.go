package logging

import (
	"fmt"
	"os"

	logging "github.com/op/go-logging"
)

// Example format string. Everything except the message has a custom color
// which is dependent on the log level. Many fields have a custom output
// formatting too, eg. the time returns the hour down to the milli second.
var (
	formatter = logging.MustStringFormatter(
		"%{color}" + getLogHostname() +
			"%{shortfile}:%{shortfunc} :: %{level:.4s} %{id:03x}%{color:reset} %{message}",
	)
	DefaultLogLevel = logging.INFO
)

// getLogHostname gets the string for logging the hostname if the "HOSTNAME" variable exists
func getLogHostname() string {
	hostname := os.Getenv("HOSTNAME")
	if hostname != "" {
		return fmt.Sprintf("[%s] ", hostname)
	} else {
		return ""
	}
}

// GetLogger takes a module name and an option override to create a go-logging logging handler
func GetLogger(name string, levelOverride ...string) *logging.Logger {
	var err error

	logLevel := DefaultLogLevel
	if len(levelOverride) > 0 {
		overrideLevelName := levelOverride[0]
		logLevel, err = logging.LogLevel(overrideLevelName)
		if err != nil {
			// Can't use logging yet, since it may be not be setup yet
			fmt.Println("Error getting log level `%s`. Err: %v", overrideLevelName, err)
			logLevel = DefaultLogLevel
		}
	}

	l := logging.MustGetLogger(name)
	logging.SetLevel(logLevel, "")
	logging.SetFormatter(formatter)
	return l
}
