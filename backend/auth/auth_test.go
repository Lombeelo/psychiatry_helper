package auth_test

import (
	"net/http/httptest"
	"testing"

	"example.com/backend/auth"
	"example.com/backend/domain"
)

func TestJWTTokenRoundTrip(t *testing.T) {
	authenticator := auth.NewJWT("test-secret")
	token, err := authenticator.IssueToken("demo-doctor", domain.RoleDoctor)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	request := httptest.NewRequest("GET", "/", nil)
	request.Header.Set("Authorization", "Bearer "+token)

	principal, err := authenticator.Authenticate(request)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if principal.UserID != "demo-doctor" || principal.Role != domain.RoleDoctor {
		t.Fatalf("unexpected principal: %+v", principal)
	}
}

func TestJWTRejectsMissingToken(t *testing.T) {
	authenticator := auth.NewJWT("test-secret")
	request := httptest.NewRequest("GET", "/", nil)

	if _, err := authenticator.Authenticate(request); err == nil {
		t.Fatal("expected authentication error")
	}
}
