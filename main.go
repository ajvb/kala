package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/ajvb/kala/api"

	"github.com/222Labs/common/go/logging"
	"github.com/codegangsta/cli"
	"github.com/gorilla/mux"
)

var (
	log = logging.GetLogger("kala")
	// TODO - fix
	staticDir = "/home/ajvb/Code/kala/ui/static"
)

func initServer() *mux.Router {
	r := mux.NewRouter()
	// API
	api.SetupApiRoutes(r)

	return r
}

func main() {
	app := cli.NewApp()
	app.Name = "Kala"
	app.Usage = "Modern job scheduler"
	app.Version = "0.1"
	app.Commands = []cli.Command{
		{
			Name:  "run",
			Usage: "run kala",
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  "port, p",
					Value: 8000,
					Usage: "port for Kala to run on",
				},
				cli.BoolFlag{
					Name:  "debug, d",
					Usage: "debug logging",
				},
			},
			Action: func(c *cli.Context) {
				var parsedPort string
				port := c.Int("port")
				if port != 0 {
					parsedPort = fmt.Sprintf(":%d", port)
				} else {
					parsedPort = ":8000"
				}

				// TODO set log level
				if c.Bool("debug") {
				}

				r := initServer()

				log.Info("Starting server...")
				log.Fatal(http.ListenAndServe(parsedPort, r))
			},
		},
	}

	app.Run(os.Args)
}
