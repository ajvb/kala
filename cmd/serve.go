package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/ajvb/kala/api"
	"github.com/ajvb/kala/job"
	"github.com/ajvb/kala/job/storage/boltdb"
	"github.com/ajvb/kala/job/storage/postgres"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "start kala server",
	Long:  `start the kala server with the backing store of your choice, and run until interrupted`,
	Run: func(cmd *cobra.Command, args []string) {
		if viper.GetBool("verbose") {
			log.SetLevel(log.DebugLevel)
		}

		var parsedPort string
		port := viper.GetString("port")
		if port != "" {
			if strings.Contains(port, ":") {
				parsedPort = port
			} else {
				parsedPort = ":" + port
			}
		} else {
			parsedPort = ":8000"
		}

		var connectionString string
		if viper.GetString("interface") != "" {
			connectionString = viper.GetString("interface") + parsedPort
		} else {
			connectionString = parsedPort
		}

		var db job.JobDB

		switch viper.GetString("jobdb") {
		case "boltdb":
			db = boltdb.GetBoltDB(viper.GetString("boltpath"))
		case "postgres":
			dsn := fmt.Sprintf("postgres://%s:%s@%s", viper.GetString("jobdb-username"), viper.GetString("jobdb-password"), viper.GetString("jobdb-address"))
			db = postgres.New(dsn)
		default:
			log.Fatalf("Unknown Job DB implementation '%s'", viper.GetString("jobdb"))
		}

		if viper.GetBool("no-persist") {
			log.Warn("No-persist mode engaged; using in-memory database!")
			db = &job.MockDB{}
		}

		// Create cache
		log.Infof("Preparing cache")
		cache := job.NewLockFreeJobCache(db)

		// Persistence mode
		persistEvery := viper.GetInt("persist-every")
		if viper.GetBool("no-tx-persist") {
			log.Warnf("Transactional persistence is not enabled; job cache will persist to db every %d seconds", persistEvery)
		} else {
			log.Infof("Enabling transactional persistence, plus persist all jobs to db every %d seconds", persistEvery)
			cache.PersistOnWrite = true
		}

		if persistEvery < 1 {
			log.Fatal("With transactional persistence off, you will need to set persist-every to greater than zero.")
		}

		// Startup cache
		cache.Start(time.Duration(persistEvery)*time.Second, time.Duration(viper.GetInt("jobstat-ttl"))*time.Minute)

		// Launch API server
		log.Infof("Starting server on port %s", connectionString)
		srv := api.MakeServer(
			connectionString,
			cache,
			viper.GetString("default-owner"),
			viper.GetBool("profile"),
			viper.GetBool("no-delete-all"),
			viper.GetBool("no-local-jobs"),
		)
		log.Fatal(srv.ListenAndServe())
	},
}

func init() {
	RootCmd.AddCommand(serveCmd)
	serveCmd.Flags().StringP("port", "p", ":8000", "Port for Kala to run on.")
	serveCmd.Flags().BoolP("no-persist", "n", false, "No Persistence Mode - In this mode no data will be saved to the database. Perfect for testing.")
	serveCmd.Flags().StringP("interface", "i", "", "Interface to listen on, default is all.")
	serveCmd.Flags().StringP("default-owner", "o", "", "Default owner. The inputted email will be attached to any job missing an owner")
	serveCmd.Flags().String("jobdb", "boltdb", "Implementation of job database, either 'boltdb' or 'postgres'.")
	serveCmd.Flags().String("bolt-path", "", "Path to the bolt database file, default is current directory.")
	serveCmd.Flags().String("jobdb-address", "", "Network address for the job database, in 'host:port' format.")
	serveCmd.Flags().String("jobdb-username", "", "Username for the job database.")
	serveCmd.Flags().String("jobdb-password", "", "Password for the job database.")
	serveCmd.Flags().BoolP("verbose", "v", false, "Set for verbose logging.")
	serveCmd.Flags().IntP("persist-every", "e", 60*60, "Interval in seconds between persisting all jobs to db") //nolint:gomnd
	serveCmd.Flags().Int("jobstat-ttl", -1, "Sets the jobstat-ttl in minutes. The default -1 value indicates JobStat entries will be kept forever")
	serveCmd.Flags().Bool("profile", false, "Activate pprof handlers")
	serveCmd.Flags().Bool("no-tx-persist", false, "Only persist to db periodically, not transactionally.")
	serveCmd.Flags().Bool("no-delete-all", false, "Disable the delete all jobs endpoint.")
	serveCmd.Flags().Bool("no-local-jobs", false, "Disable creating local jobs via API.")
}
