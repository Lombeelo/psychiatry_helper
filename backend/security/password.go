// Package security содержит минимальные средства хеширования паролей.
// Для MVP используется SHA-256 с индивидуальной солью; в production
// следует заменить на bcrypt/argon2.
package security

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// HashPassword возвращает хеш и соль в base64.
func HashPassword(password string) (hash string, salt string, err error) {
	rawSalt := make([]byte, 16)
	if _, err := rand.Read(rawSalt); err != nil {
		return "", "", fmt.Errorf("generate password salt: %w", err)
	}

	salt = base64.RawStdEncoding.EncodeToString(rawSalt)
	sum := sha256.Sum256(append(rawSalt, []byte(password)...))
	hash = hex.EncodeToString(sum[:])
	return hash, salt, nil
}

// VerifyPassword сравнивает пароль с сохранёнными хешем и солью.
func VerifyPassword(password, hash, salt string) bool {
	rawSalt, err := base64.RawStdEncoding.DecodeString(salt)
	if err != nil {
		return false
	}
	sum := sha256.Sum256(append(rawSalt, []byte(password)...))
	expected := hex.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(expected), []byte(hash)) == 1
}
