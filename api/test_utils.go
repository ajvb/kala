package api

import (
	"net/http/httptest"
	"time"

	"github.com/ajvb/kala/job"

	"github.com/gorilla/mux"
)

func NewTestServer() *httptest.Server {
	r := mux.NewRouter()
	db := &job.MockDB{}
	cache := job.NewMemoryJobCache(db, time.Hour)
	SetupApiRoutes(r, cache, db)
	return httptest.NewServer(r)
}
