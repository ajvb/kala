package main

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/ajvb/kala/api"
	"github.com/ajvb/kala/job"
	"github.com/ajvb/kala/job/storage/boltdb"
	"github.com/ajvb/kala/utils/logging"

	"github.com/codegangsta/cli"
)

var (
	log                 = logging.GetLogger("kala")
	DefaultPersistEvery = 5 * time.Second
)

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
				cli.StringFlag{
					Name:  "interface, i",
					Value: "",
					Usage: "Interface to listen on, default is all",
				},
				cli.StringFlag{
					Name:  "boltpath",
					Value: "",
					Usage: "Path to the bolt database file, default is current directory.",
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

				var connectionString string
				if c.String("interface") != "" {
					connectionString = c.String("interface") + parsedPort
				} else {
					connectionString = parsedPort
				}

				db := boltdb.GetBoltDB(c.String("boltpath"))

				// Create cache
				cache := job.NewMemoryJobCache(db, DefaultPersistEvery)
				cache.Start()

				log.Info("Starting server on port %s...", connectionString)
				log.Fatal(api.StartServer(connectionString, cache, db))
			},
		},
	}

	app.Run(os.Args)
}
