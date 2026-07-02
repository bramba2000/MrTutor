package validation

import (
	"strings"
)

// Problem represents a validation problem with a specific field and message.
type Problem struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Builder accumulates validation problems and builds a final error. Its methods
// return the builder so calls can be chained:
//
//	return (&validation.Builder{}).
//		Field("token", validation.Required(r.Token)).
//		Field("password", validation.Required(r.Password)).
//		Err()
type Builder struct {
	problems []Problem
}

// Add records a problem for the given field and returns the builder.
func (b *Builder) Add(field, msg string) *Builder {
	b.problems = append(b.problems, Problem{
		Field:   field,
		Message: msg,
	})
	return b
}

// Field records a problem for each non-nil validator result under field and
// returns the builder. Nil results (passing validators) are ignored.
func (b *Builder) Field(field string, errs ...error) *Builder {
	for _, err := range errs {
		if err != nil {
			b.Add(field, err.Error())
		}
	}
	return b
}

// Err returns an error if there are any accumulated problems, or nil if there are none.
func (b *Builder) Err() error {
	if len(b.problems) == 0 {
		return nil
	}
	return &Error{Problems: b.problems}
}

// FieldRule pairs a field name with the results of running its validators.
// Construct one with [Field] and pass it to [Fields].
type FieldRule struct {
	name string
	errs []error
}

// Field builds a [FieldRule] for use with [Fields], grouping the given validator
// results under one field name. Nil results (passing validators) are ignored.
func Field(name string, errs ...error) FieldRule {
	return FieldRule{name: name, errs: errs}
}

// Fields runs several field rules and returns a *[Error] if any produced problems,
// or nil if all passed. It is the declarative shorthand for a [Builder] when no
// conditional logic is needed:
//
//	func (r RegisterRequest) Validate() error {
//		return validation.Fields(
//			validation.Field("username", validation.Required(r.Username)),
//			validation.Field("email", validation.Required(r.Email), validation.Email(r.Email)),
//		)
//	}
func Fields(rules ...FieldRule) error {
	b := &Builder{}
	for _, r := range rules {
		b.Field(r.name, r.errs...)
	}
	return b.Err()
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
