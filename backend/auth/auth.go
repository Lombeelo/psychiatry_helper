package auth

import (
	"errors"
	"net/http"
	"strings"

	"example.com/backend/domain"
	"example.com/backend/httpapi"
)

type DevAuthenticator struct{}

func (DevAuthenticator) Authenticate(r *http.Request) (httpapi.Principal, error) {
	userID := strings.TrimSpace(r.Header.Get("X-Dev-User-ID"))
	role := strings.TrimSpace(r.Header.Get("X-Dev-Role"))

	if userID == "" || role == "" {
		return httpapi.Principal{}, errors.New(
			"set X-Dev-User-ID and X-Dev-Role headers",
		)
	}

	return httpapi.Principal{
		UserID: domain.ID(userID),
		Role:   domain.AccountRole(role),
	}, nil
}
