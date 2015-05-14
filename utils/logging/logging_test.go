package logging

import (
	"os"

	"github.com/stretchr/testify/assert"
	"testing"
)

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
