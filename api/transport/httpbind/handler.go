// Package httpbind adapts service functions to net/http handlers by wiring
// together a decode, service call, and encode step.
package httpbind

import (
	"context"
	"encoding/json"
	"errors"
	apierrors "mrtutor-api/errors"
	"mrtutor-api/validation"
	"net/http"
)

// writeError maps domain errors to HTTP status codes and writes the error response.
func writeError(w http.ResponseWriter, err error) {
	if validationErr, ok := errors.AsType[*validation.ValidationError](err); ok {
		http.Error(w, validationErr.Error(), http.StatusBadRequest)
		return
	} else if notFoundErr, ok := errors.AsType[apierrors.NotFoundError](err); ok {
		http.Error(w, notFoundErr.Error(), http.StatusNotFound)
		return
	} else if errors.Is(err, apierrors.ErrUnauthorized) {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	} else {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Handler wires a decode → service → encode pipeline into an http.Handler.
// A decode failure produces 400; service errors are mapped by writeError; encode failures produce 500.
func Handler[In, Out any](
	decode func(*http.Request) (In, error),
	fn func(context.Context, In) (Out, error),
	encode func(http.ResponseWriter, Out) error,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		in, err := decode(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		out, err := fn(r.Context(), in)
		if err != nil {
			writeError(w, err)
			return
		}

		if err := encode(w, out); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

// NewJSONDecoder returns a decoder that reads a JSON request body into In.
func NewJSONDecoder[In any]() func(*http.Request) (In, error) {
	return func(r *http.Request) (In, error) {
		if r.Body == nil {
			return *new(In), errors.New("request body is empty")
		}
		var in In
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			return *new(In), err
		}
		return in, nil
	}
}

// NewJSONEncoder returns an encoder that writes Out as JSON with the given status code.
func NewJSONEncoder[Out any](statusCode int) func(http.ResponseWriter, Out) error {
	return func(w http.ResponseWriter, out Out) error {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		return json.NewEncoder(w).Encode(out)
	}
}
