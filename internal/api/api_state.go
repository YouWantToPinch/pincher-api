package api

import (
	"net/http"
)

func endpReadiness(w http.ResponseWriter, r *http.Request) {
	respondWithText(w, http.StatusOK, "OK")
}
