package api

import (
	"net/http"
)

func (c *APIConfig) handleReadiness(w http.ResponseWriter, r *http.Request) {
	respondWithText(w, http.StatusOK, "OK")
}
