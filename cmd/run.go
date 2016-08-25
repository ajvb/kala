package cmd

import (
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/ajvb/kala/api"
	"github.com/ajvb/kala/job"
	"github.com/ajvb/kala/job/storage/boltdb"
	"github.com/ajvb/kala/job/storage/redis"
	"github.com/ajvb/kala/utils"
	redislib "github.com/garyburd/redigo/redis"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// DefaultPersistEvery is an interval
	DefaultPersistEvery = 5 * time.Second

	db job.JobDB
)

var RunserverCmd = &cobra.Command{
	Use:   "run",
	Short: "A brief description of your command",
	Long:  `A loooooooonger description of your command.`,
	Run: func(cmd *cobra.Command, args []string) {
		if viper.GetBool("verbose") {
			log.SetLevel(log.DebugLevel)
		}

		var parsedPort string
		port := viper.GetInt("port")
		if port != 0 {
			parsedPort = fmt.Sprintf(":%d", port)
		} else {
			parsedPort = ":8000"
		}

		var connectionString string
		if viper.GetString("interface") != "" {
			connectionString = viper.GetString("interface") + parsedPort
		} else {
			connectionString = parsedPort
		}

		switch viper.GetString("jobDB") {
		case "boltdb":
			db = boltdb.GetBoltDB(viper.GetString("boltpath"))
		case "redis":
			if viper.GetString("jobDBPassword") != "" {
				option := redislib.DialPassword(viper.GetString("jobDBPassword"))
				db = redis.New(viper.GetString("jobDBAddress"), option)
			} else {
				db = redis.New(viper.GetString("jobDBAddress"), redislib.DialOption{})
			}
		default:
			log.Fatalf("Unknown Job DB implementation '%s'", viper.GetString("jobDB"))
		}

		if viper.GetBool("no-persist") {
			db = &job.MockDB{}
		}

		// Create cache
		cache := job.NewMemoryJobCache(db)
		cache.Start(DefaultPersistEvery)

		log.Infof("Starting server on port %s...", connectionString)
		log.Fatal(api.StartServer(connectionString, cache, db, viper.GetString("default-owner")))

	},
}

func init() {
	RootCmd.AddCommand(RunserverCmd)
	utils.CreateShortBoolFlag(RunserverCmd, "verbose output", "verbose", "v", true)
	utils.CreateIntFlag(RunserverCmd, "Port to run Application server on", "port", "p", 8000)
	utils.CreateStringFlag(RunserverCmd, "Interface to listen on, default is all.", "interface", "i", "0.0.0.0")
	utils.CreateStringFlag(RunserverCmd, "Default owner. The inputted email will be attached to any job missing an owner", "default-owner", "o", "")
	utils.CreateStringFlag(RunserverCmd, "Implementation of job database, either 'boltdb' or 'redis'.", "jobDB", "", "boltdb")
	utils.CreateStringFlag(RunserverCmd, "Path to the bolt database file, default is current directory.", "boltpath", "", "")
	utils.CreateStringFlag(RunserverCmd, "Network address for the job database, in 'host:port' format.", "jobDBAddress", "", "127.0.0.1:6379")
	utils.CreateStringFlag(RunserverCmd, "Password for the job database, in 'password' format.", "jobDBPassword", "", "127.0.0.1:6379")
}
