package validation_test

import (
	"mrtutor/api/validation"
	"strings"
	"testing"
)

func TestBuilder(t *testing.T) {
	t.Run("Empty builder return nil error", func(t *testing.T) {
		builder := validation.Builder{}
		if err := builder.Err(); err != nil {
			t.Errorf("Expected nil error, got %v", err)
		}
	})
	t.Run("Builder with errors returns the first error", func(t *testing.T) {
		builder := validation.Builder{}
		builder.Add("field", "error message")
		err := builder.Err()
		if err == nil {
			t.Errorf("Expected an error, got nil")
		}
		message := err.Error()
		if !strings.Contains(message, "field") || !strings.Contains(message, "error message") {
			t.Errorf("Expected error message to contain 'field' and 'error message', got %v", message)
		}
	})

	t.Run("Builder with empty field message returns the error message", func(t *testing.T) {
		builder := validation.Builder{}
		builder.Add("", "error message")
		err := builder.Err()
		if err == nil {
			t.Errorf("Expected an error, got nil")
		}
		message := err.Error()
		if !strings.Contains(message, "error message") {
			t.Errorf("Expected error message to contain 'error message', got %v", message)
		}
	})

	t.Run("Builder with multiple message returns all error messages", func(t *testing.T) {
		builder := validation.Builder{}
		builder.Add("field1", "errors field 1")
		builder.Add("field2", "errors field 2")
		err := builder.Err()
		if err == nil {
			t.Errorf("Expected an error, got nil")
		}
		message := err.Error()
		if !strings.Contains(message, "field1") || !strings.Contains(message, "errors field 1") {
			t.Errorf("Expected error message to contain 'field1' and 'errors field 1', got %v", message)
		}
		if !strings.Contains(message, "field2") || !strings.Contains(message, "errors field 2") {
			t.Errorf("Expected error message to contain 'field2' and 'errors field 2', got %v", message)
		}
	})

	t.Run("Builder with repeated field return all messages", func(t *testing.T) {
		builder := validation.Builder{}
		builder.Add("field", "first error message")
		builder.Add("field", "second error message")
		err := builder.Err()
		if err == nil {
			t.Errorf("Expected an error, got nil")
		}
		message := err.Error()
		if !strings.Contains(message, "field") || !strings.Contains(message, "second error message") {
			t.Errorf("Expected error message to contain 'field' and 'second error message', got %v", message)
		}
		if !strings.Contains(message, "first error message") {
			t.Errorf("Expected error message to contain 'first error message', got %v", message)
		}
	})
}
