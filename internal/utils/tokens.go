package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/oklog/ulid/v2"
)

type RefreshToken struct {
	Token     string
	ExpiresAt time.Time
}

func GenerateRefreshToken() (RefreshToken, error) {
	days, err := strconv.Atoi(os.Getenv("REFRESH_TOKEN_EXPIRATION_DAYS"))
	if err != nil {
		return RefreshToken{}, fmt.Errorf("non-numeric env value for REFRESH_TOKEN_EXPIRATION_DAYS")
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return RefreshToken{}, fmt.Errorf("error generating random bytes: %w", err)
	}
	return RefreshToken{
		// combine a ULID (for uniqueness and sortability) with the random bytes (encoded in hex).
		Token:     ulid.Make().String() + hex.EncodeToString(buf),
		ExpiresAt: time.Now().Add(time.Hour * 24 * time.Duration(days)),
	}, nil
}

type jwtClaims struct {
	UserID string `json:"userID"`
	jwt.RegisteredClaims
}

func GenerateJWTAccessToken(userID string) (string, error) {
	minutes, err := strconv.Atoi(os.Getenv("ACCESS_TOKEN_EXPIRATION_MINUTES"))
	if err != nil {
		return "", fmt.Errorf("non-numeric env value for ACCESS_TOKEN_EXPIRATION_MINUTES")
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(minutes) * time.Minute)),
		},
	})
	return jwtToken.SignedString([]byte(os.Getenv("SECRET")))
}

// ParseJWTTokenString parses a JWT token string and returns its claims.
// Returns an error if the token is malformed, has an invalid signature, or uses an unexpected signing method.
func ParseJWTTokenString(tokenString string) (*jwtClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Name {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(os.Getenv("SECRET")), nil
	})
	if err != nil {
		return nil, jwt.ErrTokenSignatureInvalid
	}
	claims, ok := token.Claims.(*jwtClaims)
	if !ok {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}
