package validation

import (
	"strings"
)

// Problem represents a validation problem with a specific field and message.
type Problem struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Builder is used to accumulate validation problems and build a final error.
type Builder struct {
	problems []Problem
}

func (b *Builder) Add(field, msg string) {
	b.problems = append(b.problems, Problem{
		Field:   field,
		Message: msg,
	})
}

func (b *Builder) Field(field string, errors ...error) {
	if len(errors) == 0 {
		return
	}
	for _, error := range errors {
		if error != nil {
			b.Add(field, error.Error())
		}
	}
}

// Err returns an error if there are any accumulated problems, or nil if there are none.
func (b *Builder) Err() error {
	if len(b.problems) == 0 {
		return nil
	}
	return &Error{Problems: b.problems}
}

type Error struct {
	Problems []Problem `json:"problems"`
}

func (e *Error) Error() string {
	parts := make([]string, len(e.Problems))
	for i, p := range e.Problems {
		if p.Field != "" {
			parts[i] = p.Field + ": " + p.Message
		} else {
			parts[i] = p.Message
		}
	}
	return "validation error: " + strings.Join(parts, "; ")
}
