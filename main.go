package main

import (
	"net/http"

	"./api"
	//"./ui"

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
	api.SetupApiRoutes(r)

	// UI
	//r.HandleFunc("/", ui.HandleDashboard).Methods("GET")
	//fileServer := http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir)))
	//r.PathPrefix("/").Handler(fileServer)

	log.Info("Starting server...")
	log.Fatal(http.ListenAndServe(":8080", r))
}
