package cmd

import (
	"os"
	"strings"

	"bitbucket.org/nextiva/nextkala/api"
	"bitbucket.org/nextiva/nextkala/job"
	"bitbucket.org/nextiva/nextkala/job/storage/postgres"

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
		case "postgres":
			dsn := viper.GetString("pg-dsn")
			db = postgres.New(dsn)
		default:
			log.Fatalf("Unknown Job DB implementation '%s'", viper.GetString("jobdb"))
		}

		// Create cache
		log.Infof("Preparing cache")
		cache := job.NewLockFreeJobCache(db)

		// Startup cache
		cache.Start()

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
	if os.Getenv("SENDGRID_API_KEY") == "" {
		log.Fatal("SENDGRID_API_KEY is not set - can not start server")
	}

	RootCmd.AddCommand(serveCmd)
	serveCmd.Flags().StringP("port", "p", ":8000", "Port for Kala to run on.")
	serveCmd.Flags().StringP("interface", "i", "", "Interface to listen on, default is all.")
	serveCmd.Flags().StringP("default-owner", "o", "", "Default owner. The inputted email will be attached to any job missing an owner")
	serveCmd.Flags().String("jobdb", "postgres", "Implementation of job database.  Anything you want as long as its 'postgres'.")
	serveCmd.Flags().String("pg-dsn", "", "PostgreSQL DSN to use.")
	serveCmd.Flags().BoolP("verbose", "v", false, "Set for verbose logging.")
	serveCmd.Flags().Bool("profile", false, "Activate pprof handlers")
	serveCmd.Flags().Bool("no-delete-all", false, "Disable the delete all jobs endpoint.")
	serveCmd.Flags().Bool("no-local-jobs", false, "Disable creating local jobs via API.")
}
