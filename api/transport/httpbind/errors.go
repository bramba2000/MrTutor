package httpbind

import (
	"encoding/json"
	"errors"
	apierrors "mrtutor/api/errors"
	"mrtutor/api/validation"
	"net/http"
)

// writeJSON writes body as a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// writeError maps domain errors to HTTP status codes and writes the error response.
// Add new mappings here as guard clauses; it defaults to 500 Internal Server Error
// for unhandled errors.
func writeError(w http.ResponseWriter, err error) {
	if validationErr, ok := errors.AsType[*validation.Error](err); ok {
		// Serialize the structured problems so clients can render per-field errors.
		writeJSON(w, http.StatusBadRequest, validationErr)
		return
	}
	if errors.Is(err, ErrEmptyRequestBody) || errors.Is(err, ErrFailedToParseRequestBody) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if jsonInvalid, ok := errors.AsType[*json.SyntaxError](err); ok {
		http.Error(w, jsonInvalid.Error(), http.StatusBadRequest)
		return
	}
	if notFoundErr, ok := errors.AsType[apierrors.NotFoundError](err); ok {
		http.Error(w, notFoundErr.Error(), http.StatusNotFound)
		return
	}
	if errors.Is(err, apierrors.ErrUnauthorized) {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}
	if errors.Is(err, apierrors.ErrForbidden) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	if errors.Is(err, ErrUnacceptableContentType) {
		http.Error(w, "Unacceptable content type", http.StatusNotAcceptable)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
