package auth

import (
	"time"
	//"log"
	"errors"
	"strings"

	"net/http"
	"crypto/rand"
	"encoding/hex"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"github.com/google/uuid"
)

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 13)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func CheckPasswordHash(password, hash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		return err
	}
	return nil
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	claims := jwt.RegisteredClaims{
		Issuer:		"pincher",
		IssuedAt:	jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt:	jwt.NewNumericDate(time.Now().Add(expiresIn).UTC()),
		Subject:	userID.String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", err
	}

	return signed, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	jwtClaims := jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims, func(token *jwt.Token) (any, error) {
		return []byte(tokenSecret), nil
	})
	if err != nil {
		return uuid.Nil, err
	} else if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok {
		//log.Printf("Returning ID: %s", claims.Subject)
		id, err := uuid.Parse(claims.Subject)
		if err != nil {
			return uuid.Nil, err
		}
		return id, nil
	} else {
		return uuid.Nil, errors.New("unknown claims type, cannot proceed")
	}
}


func GetBearerToken(headers http.Header) (tokenString string, returnErr error) {
	authSlice, ok := headers["Authorization"]
	if !ok || len(authSlice) == 0 {
		return "", errors.New("Authorization header missing or empty")
	}
	authHeaderVal := authSlice[0]
	if !strings.HasPrefix(strings.ToLower(authHeaderVal), "bearer ") {
		return "", errors.New("No token string found")
	}
	tokenElements := strings.SplitN(authHeaderVal, " ", 2)
	if len(tokenElements) != 2 || strings.TrimSpace(tokenElements[1]) == "" {
		return "", errors.New("Bearer presented without token")
	}
	return tokenElements[1], nil
}

func MakeRefreshToken() (string, error) {
	rBytes := make([]byte, 32)
	_, err := rand.Read(rBytes)
	if err != nil {
		return "", err
	}
	hexString := hex.EncodeToString(rBytes)
	
	return hexString, nil
}

func GetAPIKey(headers http.Header) (string, error) {
	authSlice, ok := headers["Authorization"]
	if !ok || len(authSlice) == 0 {
		return "", errors.New("Authorization header missing or empty")
	}
	authHeaderVal := authSlice[0]
	if !strings.HasPrefix(strings.ToLower(authHeaderVal), "apikey ") {
		return "", errors.New("No key string found")
	}
	tokenElements := strings.SplitN(authHeaderVal, " ", 2)
	if len(tokenElements) != 2 || strings.TrimSpace(tokenElements[1]) == "" {
		return "", errors.New("ApiKey string missing")
	}
	return tokenElements[1], nil
}