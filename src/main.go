package main

import (
	"net/http"

	"./api"

	"github.com/222Labs/common/go/logging"
	"github.com/gorilla/mux"
)

var (
	log = logging.GetLogger("kala")
)

func main() {
	r := mux.NewRouter()

	apiUrlPrefix := "/api/v1/job/"
	// Route for creating a job
	r.HandleFunc(apiUrlPrefix, api.HandleAddJob).Methods("POST")
	// Route for deleting and getting a job
	r.HandleFunc(apiUrlPrefix+"{id}", api.HandleJobRequest).Methods("DELETE", "GET")
	// Route for listing all jops
	r.HandleFunc(apiUrlPrefix, api.HandleListJobs).Methods("GET")
	// Route for manually start a job
	r.HandleFunc(apiUrlPrefix+"start/{id}", api.HandleStartJobRequest).Methods("POST")

	log.Info("Starting server...")

	log.Fatal(http.ListenAndServe(":8000", r))
}
