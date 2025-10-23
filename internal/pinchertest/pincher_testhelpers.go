package pinchertest

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
)

func GetJSONField(w *httptest.ResponseRecorder, field string) (any, error) {
	res := w.Result()
	defer res.Body.Close()

	var body map[string]any
	decoder := json.NewDecoder(res.Body)
	decoder.UseNumber()
	err := decoder.Decode(&body)
	if err != nil {
		return nil, err
	}
	val, ok := body[field]
	if !ok {
		return nil, fmt.Errorf("field %s not found in response", field)
	}

	if num, ok := val.(json.Number); ok {
		if i, err := num.Int64(); err == nil {
			return i, nil
		}
		if f, err := num.Float64(); err == nil {
			return f, nil
		}
	}

	return val, nil
}
