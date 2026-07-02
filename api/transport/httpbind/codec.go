package httpbind

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
)

var (
	ErrEmptyRequestBody         = errors.New("request body is empty")
	ErrUnacceptableContentType  = errors.New("cannot decode request body: unacceptable content type")
	ErrFailedToParseRequestBody = errors.New("failed to parse request body")
)

// NewJSONDecoder returns a decoder that reads a JSON request body into In.
func NewJSONDecoder[In any]() func(*http.Request) (In, error) {
	return func(r *http.Request) (In, error) {
		// Parse the media type so parameters like "; charset=utf-8" are tolerated.
		mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || mediaType != "application/json" {
			return *new(In), ErrUnacceptableContentType
		}
		var in In
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			if errors.Is(err, io.EOF) {
				return *new(In), ErrEmptyRequestBody
			}
			return *new(In), fmt.Errorf("%w: %w", ErrFailedToParseRequestBody, err)
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
