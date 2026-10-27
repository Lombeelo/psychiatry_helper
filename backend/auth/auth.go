package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"example.com/backend/domain"
	"example.com/backend/httpapi"
)

// JWT выдаёт и проверяет подписанные токены после входа по логину и паролю.
type JWT struct {
	secret []byte
	ttl    time.Duration
}

func NewJWT(secret string) *JWT {
	return &JWT{
		secret: []byte(secret),
		ttl:    24 * time.Hour,
	}
}

type claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// IssueToken создаёт токен для вошедшей учётной записи.
func (a *JWT) IssueToken(accountID domain.ID, role domain.AccountRole) (string, error) {
	now := time.Now().UTC()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		Role: string(role),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   string(accountID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(a.ttl)),
		},
	})
	return token.SignedString(a.secret)
}

// Authenticate проверяет заголовок Authorization: Bearer <token>.
func (a *JWT) Authenticate(r *http.Request) (httpapi.Principal, error) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(header, "Bearer ") {
		return httpapi.Principal{}, errors.New("authorization header is required")
	}
	raw := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	if raw == "" {
		return httpapi.Principal{}, errors.New("token is empty")
	}

	parsedClaims := &claims{}
	token, err := jwt.ParseWithClaims(
		raw,
		parsedClaims,
		func(token *jwt.Token) (any, error) {
			if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, errors.New("unexpected signing method")
			}
			return a.secret, nil
		},
	)
	if err != nil || !token.Valid {
		return httpapi.Principal{}, errors.New("invalid token")
	}
	if parsedClaims.Subject == "" || parsedClaims.Role == "" {
		return httpapi.Principal{}, errors.New("token has no subject or role")
	}

	return httpapi.Principal{
		UserID: domain.ID(parsedClaims.Subject),
		Role:   domain.AccountRole(parsedClaims.Role),
	}, nil
}
