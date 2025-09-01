package main

import (
	"sync/atomic"
	"net/http"
	
	"github.com/YouWantToPinch/pincher-api/internal/database"
)

type apiConfig struct {
	// atomic.Int32 is a //standard-library type that allows us to 
	// safely increment and read an integer value across multiple 
	// goroutines (HTTP requests)
	fileserverHits	atomic.Int32
	db				*database.Queries
	platform		string
	secret			string
	apiKeys			*map[string]string
}

// ================= MIDDLEWARE ================= //
func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) middlewareMetricsReset(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Store(0)
		next.ServeHTTP(w, r)
	})
}