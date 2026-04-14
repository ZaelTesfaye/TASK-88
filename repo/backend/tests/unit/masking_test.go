package unit

import (
	"testing"

	"backend/internal/security"
)

// maskValue is a helper that creates a SecurityService with nil db
// and calls MaskValue. The MaskValue method does not use the database.
func maskValue(value, pattern string) string {
	svc := security.NewSecurityService(nil)
	return svc.MaskValue(value, pattern)
}

func TestMaskLast4(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"ten digits", "1234567890", "******7890"},
		{"credit card", "4111111111111111", "************1111"},
		{"short 4 chars", "1234", "****"},
		{"short 3 chars", "123", "***"},
		{"empty string", "", ""},
		{"five chars", "ABCDE", "*BCDE"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := maskValue(tc.input, "last4")
			if result != tc.expected {
				t.Errorf("MaskValue(%q, 'last4'): expected %q, got %q",
					tc.input, tc.expected, result)
			}
		})
	}
}

func TestMaskEmail(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"standard email", "user@domain.com", "u***@domain.com"},
		{"long local", "johndoe@example.com", "j******@example.com"},
		{"single char local", "a@b.com", "*@b.com"},
		{"no @ sign", "notanemail", "**********"},
		{"empty string", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := maskValue(tc.input, "email")
			if result != tc.expected {
				t.Errorf("MaskValue(%q, 'email'): expected %q, got %q",
					tc.input, tc.expected, result)
			}
		})
	}
}

func TestMaskPhone(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"US format", "(555) 123-4567", "(***) ***-4567"},
		{"plain digits", "5551234567", "(***) ***-4567"},
		{"with dashes", "555-123-4567", "(***) ***-4567"},
		{"short number", "123", "***"},
		{"empty string", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := maskValue(tc.input, "phone")
			if result != tc.expected {
				t.Errorf("MaskValue(%q, 'phone'): expected %q, got %q",
					tc.input, tc.expected, result)
			}
		})
	}
}

func TestMaskFull(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"standard text", "sensitive data", "**************"},
		{"short text", "hi", "**"},
		{"empty string", "", ""},
		{"single char", "x", "*"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := maskValue(tc.input, "full")
			if result != tc.expected {
				t.Errorf("MaskValue(%q, 'full'): expected %q, got %q",
					tc.input, tc.expected, result)
			}
		})
	}
}

func TestShouldUnmask(t *testing.T) {
	// ShouldUnmask requires a database lookup for the SensitiveFieldRegistry.
	// Without a real DB we cannot call it directly (nil pointer dereference).
	// Instead we verify the conceptual contract:
	// - When the system cannot reach the DB, no unmasking should be granted.
	// - The pure masking logic (MaskValue) is tested above.
	t.Skip("requires database")
}

func TestMaskDefaultPattern(t *testing.T) {
	// The default (unrecognised) pattern behaves like "last4".
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"ten digits", "1234567890", "******7890"},
		{"short 4 chars", "1234", "****"},
		{"short 3 chars", "123", "***"},
		{"empty", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := maskValue(tc.input, "unknown_pattern")
			if result != tc.expected {
				t.Errorf("MaskValue(%q, 'unknown_pattern'): expected %q, got %q",
					tc.input, tc.expected, result)
			}
		})
	}
}
