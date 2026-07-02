package httpbind

import (
	"encoding/json"
	"errors"
	"net/http"
)

var (
	ErrEmptyRequestBody        = errors.New("request body is empty")
	ErrUnacceptableContentType = errors.New("cannot decode request body: unacceptable content type")
)

// NewJSONDecoder returns a decoder that reads a JSON request body into In.
func NewJSONDecoder[In any]() func(*http.Request) (In, error) {
	return func(r *http.Request) (In, error) {
		if r.Header.Get("Content-Type") != "application/json" {
			return *new(In), ErrUnacceptableContentType
		}
		if r.Body == nil {
			return *new(In), ErrEmptyRequestBody
		}
		var in In
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			return *new(In), err
		}
		return in, nil
	}
}

// NewJSONEncoder returns an encoder that writes a JSON response body from Out with the given status code.
func NewJSONEncoder[Out any](statusCode int) func(http.ResponseWriter, Out) error {
	return func(w http.ResponseWriter, out Out) error {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		return json.NewEncoder(w).Encode(out)
	}
}
