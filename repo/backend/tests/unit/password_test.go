package unit

import (
	"testing"
	"time"

	"backend/internal/auth"
	"backend/internal/models"
)

// ---------- password complexity validation ----------

func TestPasswordComplexityValid(t *testing.T) {
	validPasswords := []string{
		"Abcdef1234!@",
		"MyP@ssw0rd123",
		"Hello$World99",
		"Secur3&Str0ng!",
		"VeryL0ng!Pass#",
	}
	for _, pw := range validPasswords {
		t.Run(pw, func(t *testing.T) {
			if err := auth.ValidatePasswordComplexity(pw); err != nil {
				t.Errorf("expected valid password %q, got error: %v", pw, err)
			}
		})
	}
}

func TestPasswordComplexityTooShort(t *testing.T) {
	err := auth.ValidatePasswordComplexity("Ab1!")
	if err == nil {
		t.Error("expected error for password shorter than 12 characters")
	}
}

func TestPasswordComplexityNoUppercase(t *testing.T) {
	err := auth.ValidatePasswordComplexity("abcdef1234!@")
	if err == nil {
		t.Error("expected error for password without uppercase letter")
	}
}

func TestPasswordComplexityNoLowercase(t *testing.T) {
	err := auth.ValidatePasswordComplexity("ABCDEF1234!@")
	if err == nil {
		t.Error("expected error for password without lowercase letter")
	}
}

func TestPasswordComplexityNoDigit(t *testing.T) {
	err := auth.ValidatePasswordComplexity("Abcdefghijk!")
	if err == nil {
		t.Error("expected error for password without digit")
	}
}

func TestPasswordComplexityNoSymbol(t *testing.T) {
	err := auth.ValidatePasswordComplexity("Abcdefghijk1")
	if err == nil {
		t.Error("expected error for password without symbol")
	}
}

func TestPasswordComplexityExactMinLength(t *testing.T) {
	// Exactly 12 characters meeting all requirements.
	pw := "Abcdefgh1!23"
	if err := auth.ValidatePasswordComplexity(pw); err != nil {
		t.Errorf("expected valid password at exact min length, got error: %v", err)
	}
}

// ---------- password hashing round-trip ----------

func TestPasswordHashRoundTrip(t *testing.T) {
	passwords := []string{
		"Str0ng!Pass#99",
		"An0th3r$ecure!",
		"Unicode\u00e9Pass1!",
	}
	for _, pw := range passwords {
		t.Run(pw, func(t *testing.T) {
			hash, err := auth.HashPassword(pw)
			if err != nil {
				t.Fatalf("HashPassword error: %v", err)
			}
			ok, err := auth.VerifyPassword(hash, pw)
			if err != nil {
				t.Fatalf("VerifyPassword error: %v", err)
			}
			if !ok {
				t.Error("VerifyPassword should return true for correct password")
			}
		})
	}
}

func TestPasswordHashUniqueSalts(t *testing.T) {
	pw := "Str0ng!Pass#99"
	hash1, err := auth.HashPassword(pw)
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	hash2, err := auth.HashPassword(pw)
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if hash1 == hash2 {
		t.Error("two hashes of the same password should differ due to unique salts")
	}
}

// ---------- account lockout logic ----------

func TestLockoutAfterMaxAttempts(t *testing.T) {
	lockUntil := time.Now().Add(15 * time.Minute)
	user := &models.User{
		FailedAttempts: 5,
		LockedUntil:    &lockUntil,
	}
	if !auth.IsAccountLocked(user) {
		t.Error("user with 5 failed attempts and future lockUntil should be locked")
	}
}

func TestLockoutNotLockedWhenNilExpiry(t *testing.T) {
	user := &models.User{
		FailedAttempts: 3,
		LockedUntil:    nil,
	}
	if auth.IsAccountLocked(user) {
		t.Error("user with nil LockedUntil should NOT be locked")
	}
}

func TestLockoutExpired(t *testing.T) {
	expired := time.Now().Add(-1 * time.Minute)
	user := &models.User{
		FailedAttempts: 5,
		LockedUntil:    &expired,
	}
	if auth.IsAccountLocked(user) {
		t.Error("user whose lockout expired should NOT be locked")
	}
}

func TestLockoutStillActive(t *testing.T) {
	active := time.Now().Add(10 * time.Minute)
	user := &models.User{
		FailedAttempts: 5,
		LockedUntil:    &active,
	}
	if !auth.IsAccountLocked(user) {
		t.Error("user whose lockout is still in the future SHOULD be locked")
	}
}

func TestLockoutZeroAttempts(t *testing.T) {
	user := &models.User{
		FailedAttempts: 0,
		LockedUntil:    nil,
	}
	if auth.IsAccountLocked(user) {
		t.Error("user with 0 failed attempts should NOT be locked")
	}
}
