package main

import (
	"log"
	"fmt"
	"net/http"
)

func endpReadiness(w http.ResponseWriter, r *http.Request) {
	respondWithText(w, http.StatusOK, "OK")
}

func(cfg *apiConfig) endpFileserverHitCountGet(w http.ResponseWriter, r *http.Request) {
	output := fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Pincher Admin</h1>
    <p>Pincher has been visited %d times!</p>
  </body>
</html>`, cfg.fileserverHits.Load())

	respondWithHTML(w, http.StatusOK, output)
}

func(cfg *apiConfig) endpFileserverHitCountReset(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(fmt.Sprintf("Reset hits to: %d", cfg.fileserverHits.Load()))); err != nil {
		log.Print(err)
	}
}