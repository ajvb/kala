package main

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/ajvb/kala/api"
	"github.com/ajvb/kala/job"
	"github.com/ajvb/kala/job/storage/boltdb"
	"github.com/ajvb/kala/job/storage/consul"
	"github.com/ajvb/kala/job/storage/mongo"
	"github.com/ajvb/kala/job/storage/redis"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	redislib "github.com/garyburd/redigo/redis"
	"gopkg.in/mgo.v2"
)

func init() {
	log.SetLevel(log.WarnLevel)
}

// The current version of kala
var Version = "0.1"

func main() {
	var db job.JobDB
	runtime.GOMAXPROCS(runtime.NumCPU())

	app := cli.NewApp()
	app.Name = "Kala"
	app.Usage = "Modern job scheduler"
	app.Version = Version
	app.Commands = []cli.Command{
		{
			Name:  "run_command",
			Usage: "Run a command as if it was being run by Kala",
			Action: func(c *cli.Context) {
				if len(c.Args()) == 0 {
					log.Fatal("Must include a command")
				} else if len(c.Args()) > 1 {
					log.Fatal("Must only include a command")
				}

				cmd := c.Args()[0]

				j := &job.Job{
					Command: cmd,
				}

				err := j.RunCmd()
				if err != nil {
					log.Fatalf("Command Failed with err: %s", err)
				} else {
					fmt.Println("Command Succeeded!")
				}
			},
		},
		{
			Name:  "run",
			Usage: "run kala",
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  "port, p",
					Value: 8000,
					Usage: "Port for Kala to run on.",
				},
				cli.BoolFlag{
					Name:  "no-persist, np",
					Usage: "No Persistence Mode - In this mode no data will be saved to the database. Perfect for testing.",
				},
				cli.StringFlag{
					Name:  "interface, i",
					Value: "",
					Usage: "Interface to listen on, default is all.",
				},
				cli.StringFlag{
					Name:  "default-owner, do",
					Value: "",
					Usage: "Default owner. The inputted email will be attached to any job missing an owner",
				},
				cli.StringFlag{
					Name:  "jobDB",
					Value: "boltdb",
					Usage: "Implementation of job database, either 'boltdb', 'redis', 'mongo', or 'consul'.",
				},
				cli.StringFlag{
					Name:  "boltpath",
					Value: "",
					Usage: "Path to the bolt database file, default is current directory.",
				},
				cli.StringFlag{
					Name:  "jobDBAddress",
					Value: "",
					Usage: "Network address for the job database, in 'host:port' format.",
				},
				cli.StringFlag{
					Name:  "jobDBUsername",
					Value: "",
					Usage: "Username for the job database, in 'username' format. Currently only needed for Mongo.",
				},
				cli.StringFlag{
					Name:  "jobDBPassword",
					Value: "",
					Usage: "Password for the job database, in 'password' format.",
				},
				cli.BoolFlag{
					Name:  "verbose, v",
					Usage: "Set for verbose logging.",
				},
				cli.IntFlag{
					Name:  "persist-every",
					Value: 5,
					Usage: "Sets the persisWaitTime in seconds",
				},
				cli.IntFlag{
					Name:  "jobstat-ttl",
					Value: -1,
					Usage: "Sets the jobstat-ttl in minutes. The default -1 value indicates JobStat entries will be kept forever",
				},
			},
			Action: func(c *cli.Context) {
				if c.Bool("v") {
					log.SetLevel(log.DebugLevel)
				}

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

				switch c.String("jobDB") {
				case "boltdb":
					db = boltdb.GetBoltDB(c.String("boltpath"))
				case "redis":
					if c.String("jobDBPassword") != "" {
						option := redislib.DialPassword(c.String("jobDBPassword"))
						db = redis.New(c.String("jobDBAddress"), option, true)
					} else {
						db = redis.New(c.String("jobDBAddress"), redislib.DialOption{}, false)
					}
				case "mongo":
					if c.String("jobDBUsername") != "" {
						cred := &mgo.Credential{
							Username: c.String("jobDBUsername"),
							Password: c.String("jobDBPassword")}
						db = mongo.New(c.String("jobDBAddress"), cred)
					} else {
						db = mongo.New(c.String("jobDBAddress"), &mgo.Credential{})
					}
				case "consul":
					db = consul.New(c.String("jobDBAddress"))
				default:
					log.Fatalf("Unknown Job DB implementation '%s'", c.String("jobDB"))
				}

				if c.Bool("no-persist") {
					db = &job.MockDB{}
				}

				// Create cache
				cache := job.NewLockFreeJobCache(db)
				log.Infof("Preparing cache")
				cache.Start(time.Duration(c.Int("persist-every"))*time.Second, time.Duration(c.Int("jobstat-ttl"))*time.Minute)

				log.Infof("Starting server on port %s", connectionString)
				log.Fatal(api.StartServer(connectionString, cache, db, c.String("default-owner"), Version))
			},
		},
	}

	app.Run(os.Args)
}
