package main

import (
	"net/http"

	"./api"
	"./ui"

	"github.com/222Labs/common/go/logging"
	"github.com/gorilla/mux"
)

var (
	log       = logging.GetLogger("kala")
	staticDir = "/home/ajvb/Code/kala/src/ui/static"
)

func main() {
	r := mux.NewRouter()

	// API
	apiUrlPrefix := "/api/v1/job/"
	// Route for creating a job
	r.HandleFunc(apiUrlPrefix, api.HandleAddJob).Methods("POST")
	// Route for deleting and getting a job
	r.HandleFunc(apiUrlPrefix+"{id}", api.HandleJobRequest).Methods("DELETE", "GET")
	// Route for listing all jops
	r.HandleFunc(apiUrlPrefix, api.HandleListJobs).Methods("GET")
	// Route for manually start a job
	r.HandleFunc(apiUrlPrefix+"start/{id}", api.HandleStartJobRequest).Methods("POST")
	// Route for getting app-level metrics
	r.HandleFunc("/api/v1/stats", api.HandleKalaStats).Methods("GET")

	// UI
	r.HandleFunc("/", ui.HandleDashboard).Methods("GET")
	fileServer := http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir)))
	r.PathPrefix("/").Handler(fileServer)

	log.Info("Starting server...")
	log.Fatal(http.ListenAndServe(":8080", r))
}
