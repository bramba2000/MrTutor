// Package httpbind adapts service functions to net/http handlers by wiring
// together a decode, service call, and encode step.
package httpbind

import (
	"context"
	"net/http"
)

type Validable interface {
	Validate() error
}

func runDecode[In any](decode func(*http.Request) (In, error), r *http.Request) (In, error) {
	in, err := decode(r)
	if err != nil {
		return *new(In), err
	}
	if validable, ok := any(in).(Validable); ok {
		if err := validable.Validate(); err != nil {
			return *new(In), err
		}
	}
	return in, nil
}

// NewHandler wires a decode → service → encode pipeline into an http.NewHandler.
//
// If the [In] type implements Validable, the Validate method is called after decoding and before calling the service function. If validation fails, a 400 response is returned.
// A decode failure produces 400; service errors are mapped by writeError; encode failures produce 500.
func NewHandler[In, Out any](
	decode func(*http.Request) (In, error),
	fn func(context.Context, In) (Out, error),
	encode func(http.ResponseWriter, Out) error,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		in, err := runDecode(decode, r)
		if err != nil {
			writeError(w, err)
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

func NewNoOutputHandler[In any](
	decode func(*http.Request) (In, error),
	fn func(context.Context, In) error,
	writer func(http.ResponseWriter) error,
) http.Handler {
	return NewHandler(
		decode,
		func(ctx context.Context, in In) (struct{}, error) { return struct{}{}, fn(ctx, in) },
		func(w http.ResponseWriter, _ struct{}) error { return writer(w) },
	)
}
