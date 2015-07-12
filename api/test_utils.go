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
	cache := job.NewMemoryJobCache(db, time.Hour*1)
	SetupApiRoutes(r, cache, db)
	return httptest.NewServer(r)
}
