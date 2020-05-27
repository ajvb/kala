package cmd

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/ajvb/kala/api"
	"github.com/ajvb/kala/job"
	"github.com/ajvb/kala/job/storage/boltdb"
	"github.com/ajvb/kala/job/storage/consul"
	"github.com/ajvb/kala/job/storage/mongo"
	"github.com/ajvb/kala/job/storage/mysql"
	"github.com/ajvb/kala/job/storage/postgres"
	"github.com/ajvb/kala/job/storage/redis"

	redislib "github.com/garyburd/redigo/redis"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/mgo.v2"
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
		case "redis":
			if viper.GetString("jobdb-password") != "" {
				option := redislib.DialPassword(viper.GetString("jobdb-password"))
				db = redis.New(viper.GetString("jobdb-address"), option, true)
			} else {
				db = redis.New(viper.GetString("jobdb-address"), redislib.DialOption{}, false)
			}
		case "mongo":
			if viper.GetString("jobdb-username") != "" {
				cred := &mgo.Credential{
					Username: viper.GetString("jobdb-username"),
					Password: viper.GetString("jobdb-password")}
				db = mongo.New(viper.GetString("jobdb-address"), cred)
			} else {
				db = mongo.New(viper.GetString("jobdb-address"), &mgo.Credential{})
			}
		case "consul":
			db = consul.New(viper.GetString("jobdb-address"))
		case "postgres":
			dsn := fmt.Sprintf("postgres://%s:%s@%s", viper.GetString("jobdb-username"), viper.GetString("jobdb-password"), viper.GetString("jobdb-address"))
			db = postgres.New(dsn)
		case "mysql", "mariadb":
			dsn := fmt.Sprintf("%s:%s@%s", viper.GetString("jobdb-username"), viper.GetString("jobdb-password"), viper.GetString("jobdb-address"))
			log.Debug("Mysql/Maria DSN: ", dsn)
			if viper.GetString("jobdb-tls-capath") != "" {
				// https://godoc.org/github.com/go-sql-driver/mysql#RegisterTLSConfig
				rootCertPool := x509.NewCertPool()
				pem, err := ioutil.ReadFile(viper.GetString("jobdb-tls-capath"))
				if err != nil {
					log.Fatal(err)
				}
				if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
					log.Fatal("Failed to append PEM.")
				}
				clientCert := make([]tls.Certificate, 0, 1)
				certs, err := tls.LoadX509KeyPair(viper.GetString("jobdb-tls-certpath"), viper.GetString("jobdb-tls-keypath"))
				if err != nil {
					log.Fatal(err)
				}
				clientCert = append(clientCert, certs)
				db = mysql.New(dsn, &tls.Config{
					RootCAs:      rootCertPool,
					Certificates: clientCert,
					ServerName:   viper.GetString("jobdb-tls-servername"),
				})
			} else {
				db = mysql.New(dsn, nil)
			}
		default:
			log.Fatalf("Unknown Job DB implementation '%s'", viper.GetString("jobdb"))
		}

		if viper.GetBool("no-persist") {
			log.Warn("No-persist mode engaged; using in-memory database!")
			db = &job.MockDB{}
		}

		// Create cache
		cache := job.NewLockFreeJobCache(db)
		log.Infof("Preparing cache")
		cache.Start(time.Duration(viper.GetInt("persist-every"))*time.Second, time.Duration(viper.GetInt("jobstat-ttl"))*time.Minute)

		log.Infof("Starting server on port %s", connectionString)
		srv := api.MakeServer(connectionString, cache, db, viper.GetString("default-owner"), viper.GetBool("profile"))
		log.Fatal(srv.ListenAndServe())
	},
}

func init() {
	RootCmd.AddCommand(serveCmd)
	serveCmd.Flags().StringP("port", "p", ":8000", "Port for Kala to run on.")
	serveCmd.Flags().BoolP("no-persist", "n", false, "No Persistence Mode - In this mode no data will be saved to the database. Perfect for testing.")
	serveCmd.Flags().StringP("interface", "i", "", "Interface to listen on, default is all.")
	serveCmd.Flags().StringP("default-owner", "o", "", "Default owner. The inputted email will be attached to any job missing an owner")
	serveCmd.Flags().String("jobdb", "boltdb", "Implementation of job database, either 'boltdb', 'redis', 'mongo', 'consul', 'postgres', 'mariadb', or 'mysql'.")
	serveCmd.Flags().String("bolt-path", "", "Path to the bolt database file, default is current directory.")
	serveCmd.Flags().String("jobdb-address", "", "Network address for the job database, in 'host:port' format.")
	serveCmd.Flags().String("jobdb-username", "", "Username for the job database.")
	serveCmd.Flags().String("jobdb-password", "", "Password for the job database.")
	serveCmd.Flags().String("jobdb-tls-capath", "", "Path to tls server CA file for the job database.")
	serveCmd.Flags().String("jobdb-tls-certpath", "", "Path to tls client cert file for the job database.")
	serveCmd.Flags().String("jobdb-tls-keypath", "", "Path to tls client key file for the job database.")
	serveCmd.Flags().String("jobdb-tls-servername", "", "Server name to verify cert for the job database.")
	serveCmd.Flags().BoolP("verbose", "v", false, "Set for verbose logging.")
	serveCmd.Flags().IntP("persist-every", "e", 0, "Sets the persisWaitTime in seconds")
	serveCmd.Flags().Int("jobstat-ttl", -1, "Sets the jobstat-ttl in minutes. The default -1 value indicates JobStat entries will be kept forever")
	serveCmd.Flags().Bool("profile", false, "Activate pprof handlers")
}
