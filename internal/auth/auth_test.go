package auth

import (
	"errors"
	"strings"
	"testing"
)

func TestGenerateAndParseToken(t *testing.T) {
	token, err := GenerateToken(42)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if claims.UserID != 42 {
		t.Fatalf("user id: got %d, want 42", claims.UserID)
	}
}

func TestParseTokenInvalid(t *testing.T) {
	cases := []string{
		"",
		"not-a-jwt",
		"Bearer xxx",
	}

	for _, tc := range cases {
		t.Run(tc, func(t *testing.T) {
			_, err := ParseToken(tc)
			if !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("expected ErrInvalidToken, got %v", err)
			}
		})
	}
}

func TestAuthHeaderFormat(t *testing.T) {
	token, err := GenerateToken(1)
	if err != nil {
		t.Fatal(err)
	}

	header := AuthScheme + " " + token
	if !strings.HasPrefix(header, "Bearer ") {
		t.Fatalf("unexpected header format: %q", header)
	}
}
