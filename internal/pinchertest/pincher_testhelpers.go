package pinchertest

import (
	"fmt"
	"encoding/json"
	//"net/http"
	"net/http/httptest"
)

func GetJSONField(w *httptest.ResponseRecorder, field string) (any, error) {
	res := w.Result()
	defer res.Body.Close()
	var body map[string]any
	err := json.NewDecoder(res.Body).Decode(&body)
	if err != nil {
		return nil, err
	}
	val, ok := body[field]
	if !ok {
		return nil, fmt.Errorf("field %s not found in response", field)
	}
	return val, nil
}