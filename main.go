package main

import (
	"fmt"
	"net/http"
	"os"
	"runtime"

	"github.com/ajvb/kala/api"
	"github.com/ajvb/kala/api/middleware"
	"github.com/ajvb/kala/utils/logging"

	"github.com/codegangsta/cli"
	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
)

var (
	log = logging.GetLogger("kala")
)

func initServer() *negroni.Negroni {
	r := mux.NewRouter()
	api.SetupApiRoutes(r)
	n := negroni.New(negroni.NewRecovery(), &middleware.Logger{log})
	n.UseHandler(r)
	return n
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

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
			},
			Action: func(c *cli.Context) {
				var parsedPort string
				port := c.Int("port")
				if port != 0 {
					parsedPort = fmt.Sprintf(":%d", port)
				} else {
					parsedPort = ":8000"
				}

				r := initServer()

				log.Info("Starting server on port %s...", parsedPort)
				log.Fatal(http.ListenAndServe(parsedPort, r))
			},
		},
	}

	app.Run(os.Args)
}
