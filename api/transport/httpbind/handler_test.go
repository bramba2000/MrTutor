package httpbind_test

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	apierrors "mrtutor/api/errors"
	"mrtutor/api/transport/httpbind"
	"mrtutor/api/validation"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type TestCase[In, Out any] struct {
	name          string
	decode        func(*http.Request) (In, error)
	fn            func(context.Context, In) (Out, error)
	encode        func(http.ResponseWriter, Out) error
	matchResponse func(*http.Response) string
}

func (tc TestCase[In, Out]) Run(t *testing.T) {
	t.Parallel()
	handler := httpbind.NewHandler(
		tc.decode,
		tc.fn,
		tc.encode,
	)
	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	resp := w.Result()
	if errMsg := tc.matchResponse(resp); errMsg != "" {
		t.Error(errMsg)
	}
}

func matchStatusCode(expected int) func(*http.Response) string {
	return func(r *http.Response) string {
		if r.StatusCode != expected {
			return fmt.Sprintf("Expected status code %d got %d", expected, r.StatusCode)
		}
		return ""
	}
}

func TestHandler(t *testing.T) {

	t.Run("Decode", func(t *testing.T) {
		tt := []TestCase[[]int, any]{
			{
				name: "Decode correctly",
				decode: func(r *http.Request) ([]int, error) {
					return []int{1, 2, 3}, nil
				},
				fn: func(ctx context.Context, in []int) (any, error) {
					return in, nil
				},
				encode: func(w http.ResponseWriter, out any) error {
					return nil
				},
				matchResponse: matchStatusCode(http.StatusOK),
			},
			{
				name: "Decode error",
				decode: func(r *http.Request) ([]int, error) {
					return nil, &json.SyntaxError{}
				},
				fn: func(ctx context.Context, in []int) (any, error) {
					return in, nil
				},
				encode: func(w http.ResponseWriter, out any) error {
					return nil
				},
				matchResponse: matchStatusCode(http.StatusBadRequest),
			},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				tc.Run(t)
			})
		}
	})

	t.Run("Encode", func(t *testing.T) {
		tt := []TestCase[any, []int]{
			{
				name: "Encode correctly",
				decode: func(r *http.Request) (any, error) {
					return 1, nil
				},
				fn: func(ctx context.Context, a any) ([]int, error) {
					return []int{}, nil
				},
				encode: func(w http.ResponseWriter, out []int) error {
					w.WriteHeader(http.StatusOK)
					return nil
				},
				matchResponse: func(r *http.Response) string {
					if r.StatusCode != http.StatusOK {
						return fmt.Sprintf("Expected 200 got %d", r.StatusCode)
					}
					return ""
				},
			},
			{
				name: "Encode error",
				decode: func(r *http.Request) (any, error) {
					return 2, nil
				},
				fn: func(ctx context.Context, a any) ([]int, error) {
					return []int{}, nil
				},
				encode: func(w http.ResponseWriter, out []int) error {
					return http.ErrBodyNotAllowed
				},
				matchResponse: func(r *http.Response) string {
					if r.StatusCode != http.StatusInternalServerError {
						return fmt.Sprintf("Expected 500 got %d", r.StatusCode)
					}
					return ""
				},
			},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				tc.Run(t)
			})
		}
	})

	t.Run("Handler error", func(t *testing.T) {
		tt := []TestCase[struct{}, struct{}]{
			{
				name:   "Validation error",
				encode: func(w http.ResponseWriter, s struct{}) error { return nil },
				decode: func(r *http.Request) (struct{}, error) { return struct{}{}, nil },
				fn: func(ctx context.Context, s struct{}) (struct{}, error) {
					return struct{}{}, &validation.Error{
						Problems: []string{
							"invalid params",
							"invalid username",
						},
					}
				},
				matchResponse: func(r *http.Response) string {
					return cmp.Or(
						matchStatusCode(http.StatusBadRequest)(r),
						func(r *http.Response) string {
							body, _ := io.ReadAll(r.Body)
							if !strings.Contains(string(body), "params") || !strings.Contains(string(body), "username") {
								return fmt.Sprintf("Expected error message to contain 'params' and 'username', got %s", string(body))
							}
							return ""
						}(r),
					)
				},
			},
			{
				name:   "Internal error",
				encode: func(w http.ResponseWriter, s struct{}) error { return nil },
				decode: func(r *http.Request) (struct{}, error) { return struct{}{}, nil },
				fn: func(ctx context.Context, s struct{}) (struct{}, error) {
					return struct{}{}, fmt.Errorf("something went wrong")
				},
				matchResponse: func(r *http.Response) string {
					return cmp.Or(
						matchStatusCode(http.StatusInternalServerError)(r),
						func(r *http.Response) string {
							body, _ := io.ReadAll(r.Body)
							if !strings.Contains(string(body), "something went wrong") {
								return fmt.Sprintf("Expected error message to contain 'something went wrong', got %s", string(body))
							}
							return ""
						}(r),
					)
				},
			},
			{
				name:   "Unauthorized error",
				encode: func(w http.ResponseWriter, s struct{}) error { return nil },
				decode: func(r *http.Request) (struct{}, error) { return struct{}{}, nil },
				fn: func(ctx context.Context, s struct{}) (struct{}, error) {
					return struct{}{}, apierrors.ErrUnauthorized
				},
				matchResponse: matchStatusCode(http.StatusUnauthorized),
			},
			{
				name:   "Not found",
				encode: func(w http.ResponseWriter, s struct{}) error { return nil },
				decode: func(r *http.Request) (struct{}, error) { return struct{}{}, nil },
				fn: func(ctx context.Context, s struct{}) (struct{}, error) {
					return struct{}{}, apierrors.NotFoundError{Entity: "resource"}
				},
				matchResponse: matchStatusCode(http.StatusNotFound),
			},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				tc.Run(t)
			})
		}
	})
}
