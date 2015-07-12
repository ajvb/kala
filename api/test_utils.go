package api

import (
	"net/http/httptest"

	"github.com/gorilla/mux"
)

func NewTestServer() *httptest.Server {
	r := mux.NewRouter()
	//SetupApiRoutes(r)
	return httptest.NewServer(r)
}
