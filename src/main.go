package main

import (
	"net/http"

	"./api"

	"github.com/222Labs/common/go/logging"
	"github.com/gorilla/mux"
)

var (
	log     = logging.GetLogger("kala")
)

func main() {
	r := mux.NewRouter()

	apiUrlPrefix := "/api/v1/job/"
	// CRUD
	r.HandleFunc(apiUrlPrefix, api.HandleAddJob).Methods("POST")
	r.HandleFunc(apiUrlPrefix+"{id}", api.HandleJobRequest).Methods("DELETE", "GET")
	r.HandleFunc(apiUrlPrefix+"list", api.HandleListJobs).Methods("GET")
	// TODO
	// Manually start a job
	// Adding a dependent job

	log.Info("Starting server...")

	log.Fatal(http.ListenAndServe(":8000", r))
}
