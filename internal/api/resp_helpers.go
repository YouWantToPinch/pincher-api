package api

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"log/slog"
	"net/http"
	"time"
)

func decodeParams[T any](r *http.Request) (T, error) {
	defer r.Body.Close()
	var v T
	err := json.NewDecoder(r.Body).Decode(&v)
	return v, err
}

func respondWithError(w http.ResponseWriter, code int, msg string, err error) {

	if err != nil {
		slog.Error(err.Error())
	}

	type errorResponse struct {
		Error string `json:"error"`
	}
	slog.Error(msg)
	respondWithJSON(w, code, errorResponse{
		Error: msg,
	})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	dat, err := json.Marshal(payload)
	if err != nil {
		slog.Error("Could not marshal JSON for response: " + err.Error())
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}

func respondWithHTML(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	if _, err := w.Write([]byte(msg)); err != nil {
		slog.Error(err.Error())
	}
}

func respondWithText(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(code)
	if _, err := w.Write([]byte(msg)); err != nil {
		slog.Error(err.Error())
	}
}

// Try to parse input path parameter; store uuid.Nil into 'parse' on failure
func parseUUIDFromPath(pathParam string, r *http.Request, parse *uuid.UUID) error {
	idString := r.PathValue(pathParam)
	if idString != "" {
		parsedID, err := uuid.Parse(idString)
		if err != nil {
			return fmt.Errorf("Parameter value '%s' for provided path parameter '%s' could not be parsed as UUID", idString, pathParam)
		}
		*parse = parsedID
	} else {
		*parse = uuid.Nil
	}
	return nil
}

// Try to parse input query parameter; store time.Time{} into 'parse' on failure
func parseDateFromQuery(queryParam string, r *http.Request, parse *time.Time) error {
	dateString := r.URL.Query().Get(queryParam)

	if dateString == "" {
		*parse = time.Time{}
		return nil
	}

	var parsedDate time.Time
	var err error

	timeLayouts := []string{
		time.RFC3339,
		"2006-01-02",
	}

	for _, layout := range timeLayouts {
		parsedDate, err = time.Parse(layout, dateString)
		if err == nil {
			*parse = parsedDate
			return nil
		}
	}

	return fmt.Errorf("Query value '%s' for provided parameter '%s' could not be parsed as DATE", dateString, queryParam)
}

// Try to parse input path parameter; store time.Time{} into 'parse' on failure
func parseDateFromPath(pathParam string, r *http.Request, parse *time.Time) error {
	dateString := r.PathValue(pathParam)
	if dateString == "" {
		*parse = time.Time{}
		return nil
	}

	var parsedDate time.Time
	var err error

	timeLayouts := []string{
		time.RFC3339,
		"2006-01-02",
	}

	for _, layout := range timeLayouts {
		parsedDate, err = time.Parse(layout, dateString)
		if err == nil {
			*parse = parsedDate
			return nil
		}
	}

	return fmt.Errorf("Path value '%s' for provided parameter '%s' could not be parsed as DATE", dateString, pathParam)
}
