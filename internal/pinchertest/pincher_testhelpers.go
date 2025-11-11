package pinchertest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
)

func MakeRequest(method, path, token string, body any) *http.Request {
	var buffer io.Reader

	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			panic(err)
		}
		buffer = bytes.NewReader(b)
	}

	req := httptest.NewRequest(method, path, buffer)
	req.Header.Set("Content-Type", "application/json")

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return req
}
