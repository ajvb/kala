package main

import (
	"os"
	"runtime"

	"github.com/ajvb/kala/cmd"

	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetLevel(log.InfoLevel)
}

// The current version of kala
var Version = "0.1"

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	cmd.App.Version = Version
	cmd.App.Run(os.Args)
}
