package api

import (
	"context"
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
	err := json.NewDecoder(r.Body).Decode(&v)
	defer r.Body.Close()
	if err != nil {
		return v, fmt.Errorf("failure decoding request payload: %w", err)
	}
	return v, err
}

func makeStatusCodeMsg(code int) string {
	return fmt.Sprintf("%d %s", code, http.StatusText(code))
}

func respondWithError(w http.ResponseWriter, code int, msg string, err error) {
	// prefix the message with a status code message
	errorMessage := makeStatusCodeMsg(code)
	// add the optional info message, if it exists
	if msg != "" {
		errorMessage += fmt.Sprintf("; %s", msg)
	}
	// add the technical error message, if it exists
	if err != nil {
		errorMessage += fmt.Sprintf(": %s", err.Error())
	}

	// log the error on the server
	slog.Error(errorMessage, slog.Int("HTTP Status Code", code))

	// respond with the errorMessage as JSON
	type errorResponse struct {
		Error string `json:"error"`
	}
	respondWithJSON(w, code, errorResponse{
		Error: msg,
	})
}

func respondWithJSON(w http.ResponseWriter, code int, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		slog.Error("could not marshal JSON for response: " + err.Error())
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, err = w.Write(data)
	if err != nil {
		slog.Error("could not write to header from JSON payload: " + err.Error())
	}
}

// respondWithCode responds with a text body including only a status code message
func respondWithCode(w http.ResponseWriter, code int) {
	switch code {
	case http.StatusNoContent:
		w.WriteHeader(code)
	default:
		respondWithText(w, code, "")
	}
}

func respondWithText(w http.ResponseWriter, code int, msg string) {
	// if message is empty, set it to AT LEAST the status code message
	if msg == "" {
		msg = makeStatusCodeMsg(code)
	}

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
			return fmt.Errorf("value '%s' for path parameter '%s' could not be parsed as UUID: %w", uuidString, pathParam, err)
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
	err := parseDate(dateString, parse)
	if err != nil {
		return fmt.Errorf("invalid query parameter value '%s': %w", queryParam, err)
	}
	return nil
}

func parseDateFromPath(pathParam string, r *http.Request, parse *time.Time) error {
	dateString := r.PathValue(pathParam)
	err := parseDate(dateString, parse)
	if err != nil {
		return fmt.Errorf("invalid path parameter value '%s': %w", pathParam, err)
	}
	return nil
}

// Try to parse input dateString according to available time layouts.
// Store time.Time{} into 'parse' on failure.
func parseDate(dateString string, parse *time.Time) error {
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

	return fmt.Errorf("value '%s' could not be parsed as DATE", dateString)
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

func lookupResourceIDByName[T any](ctx context.Context, arg T, dbQuery func(context.Context, T) (uuid.UUID, error)) (*uuid.UUID, error) {
	id, err := dbQuery(ctx, arg)
	if err != nil {
		return &uuid.Nil, err
	}
	return &id, err
}
