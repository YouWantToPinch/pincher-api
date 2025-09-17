package integration

import (
	"testing"
	"httptest"
)

const baseURL = "http://localhost:8080"

func SetupTestDB() error {
	connStr := ""
	db, err
}

func create_user(username string, password string) {
	req := httptest.NewRequest(http.MethodPost, "/api/users", nil)
	w := httptest.NewRecorder()

	// Send the database with some test data
	user := 
}