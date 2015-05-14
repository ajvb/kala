package logging

import (
	"os"

	logging "github.com/op/go-logging"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetLogLevelDefaultHasEnv(t *testing.T) {
	os.Setenv(LOGLEVEL_ENV_NAME, logging.CRITICAL.String())
	level := getLogLevelDefault(logging.INFO)
	assert.Equal(t, logging.CRITICAL, level)
}

func TestGetLogLevelDefaultBadEnv(t *testing.T) {
	os.Setenv(LOGLEVEL_ENV_NAME, "$@HJFJ")
	level := getLogLevelDefault(logging.INFO)
	assert.Equal(t, logging.INFO, level)
}

func TestGetLogLevelDefaultUnsetEnv(t *testing.T) {
	level := getLogLevelDefault(logging.WARNING)
	assert.Equal(t, logging.WARNING, level)
}

func TestGetLogHostnameEnvSet(t *testing.T) {
	hostname := "222labs.com"
	os.Setenv("HOSTNAME", hostname)
	logHostname := getLogHostname()
	assert.Contains(t, logHostname, hostname)
}

func TestGetLogHostnameEnvEmpty(t *testing.T) {
	os.Setenv("HOSTNAME", "")
	logHostname := getLogHostname()
	assert.Empty(t, logHostname)
}
func TestGetLogHostnameEnvUnset(t *testing.T) {
	os.Clearenv()
	logHostname := getLogHostname()
	assert.Empty(t, logHostname)
}
