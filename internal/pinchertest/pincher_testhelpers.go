package pinchertest

import (
	"fmt"
	"net/http"
)

// ========== MIDDLEWARE ==========

func headerJSON(req *http.Request) *http.Request {
	req.Header.Set("Content-Type", "application/json")
	return req
}

func requireToken(req *http.Request, token string) *http.Request {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	return req
}
