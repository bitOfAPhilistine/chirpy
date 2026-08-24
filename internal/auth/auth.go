package auth

import (
	"fmt"
	"time"
	"strings"
	"net/http"
	"crypto/rand"
	"encoding/hex"
	"github.com/google/uuid"
	"github.com/golang-jwt/jwt/v5"
	"github.com/alexedwards/argon2id"
)


func HashPassword(password string) (string, error) {
	return argon2id.CreateHash(password, argon2id.DefaultParams)
}

func CheckPasswordHash(password, hash string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, hash)
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer: "chirpy-access",
		IssuedAt: jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
		Subject: userID.String(),
	})

	jwt, err := token.SignedString([]byte(tokenSecret))
	if err != nil {return "", err}

	return jwt, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {return []byte(tokenSecret), nil})
	if err != nil {return uuid.UUID{}, err}

	iss, err := token.Claims.GetSubject()
	if err != nil {return uuid.UUID{}, err}

	id, err := uuid.Parse(iss)
	if err != nil {return uuid.UUID{}, err}

	return id, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	ts := headers.Get("Authorization")
	if ts == "" {return "", fmt.Errorf("no token string found in headers %v", headers)}

	return strings.Split(ts, " ")[1], nil
}

func MakeRefreshToken() string {
	token := make([]byte, 32)
	rand.Read(token)
	return hex.EncodeToString(token)
}

func GetAPIKey(headers http.Header) (string, error) {
	ts := headers.Get("Authorization")
	if ts == "" {return "", fmt.Errorf("no api key string found in headers %v", headers)}

	return strings.Split(ts, " ")[1], nil
}