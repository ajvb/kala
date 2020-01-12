package main

import (
	"os"

	"github.com/ajvb/kala/cmd"

	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetLevel(log.InfoLevel)
}

// The current version of kala
var Version = "0.1"

func main() {
	cmd.App.Version = Version
	cmd.App.Run(os.Args)
}
