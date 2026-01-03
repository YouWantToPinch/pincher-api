package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

func decodePayload[T any](r *http.Request) (T, error) {
	var v T
	decodeErr := json.NewDecoder(r.Body).Decode(&v)
	err := r.Body.Close()
	if err != nil {
		slog.Error(err.Error())
	}
	return v, decodeErr
}

func respondWithError(w http.ResponseWriter, code int, msg string, err error) {
	errorMessage := msg
	if err != nil {
		errorMessage += "; " + err.Error()
	}

	type rspSchema struct {
		Error string `json:"error"`
	}
	slog.Error(errorMessage, slog.Int("Code", code))
	respondWithJSON(w, code, rspSchema{
		Error: errorMessage,
	})
}

func respondWithJSON(w http.ResponseWriter, code int, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		slog.Error("Could not marshal JSON for response: " + err.Error())
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, err = w.Write(data)
	if err != nil {
		slog.Error("Could not write to header from JSON payload; " + err.Error())
	}
}

func respondWithCode(w http.ResponseWriter, code int) {
	w.WriteHeader(code)
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
	uuidString := r.PathValue(pathParam)
	if uuidString != "" {
		parsedID, err := uuid.Parse(uuidString)
		if err != nil {
			return fmt.Errorf("value '%s' for provided path parameter '%s' could not be parsed as UUID: %s", uuidString, pathParam, err.Error())
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
		"2006-01-02",
	}

	for _, layout := range timeLayouts {
		parsedDate, err = time.Parse(layout, dateString)
		if err == nil {
			*parse = parsedDate
			return nil
		}
	}

	return fmt.Errorf("value '%s' for provided query parameter '%s' could not be parsed as DATE", dateString, queryParam)
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
		"2006-01-02",
	}

	for _, layout := range timeLayouts {
		parsedDate, err = time.Parse(layout, dateString)
		if err == nil {
			*parse = parsedDate
			return nil
		}
	}

	return fmt.Errorf("path value '%s' for provided parameter '%s' could not be parsed as DATE", dateString, pathParam)
}

func parseBoolFromString(s string) (bool, error) {
	switch s {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, errors.New("provided string value for 'Cleared' could not be parsed; must be 'true' or 'false'")
	}
}
