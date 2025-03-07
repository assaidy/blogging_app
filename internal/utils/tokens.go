package utils

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/assaidy/blogging_app/internal/repositry"
	"github.com/gofiber/fiber/v2"
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

type claims struct {
	UserID string `json:"userID"`
	jwt.RegisteredClaims
}

func GenerateJWTAccessToken(userID string) (string, error) {
	minutes, err := strconv.Atoi(os.Getenv("ACCESS_TOKEN_EXPIRATION_MINUTES"))
	if err != nil {
		return "", fmt.Errorf("non-numeric env value for ACCESS_TOKEN_EXPIRATION_MINUTES")
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(minutes) * time.Minute)),
		},
	})
	return jwtToken.SignedString([]byte(os.Getenv("SECRET")))
}

// parseJWTTokenString parses a JWT token string and returns its claims.
// Returns an error if the token is malformed, has an invalid signature, or uses an unexpected signing method.
func parseJWTTokenString(tokenString string) (*claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &claims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Name {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(os.Getenv("SECRET")), nil
	})
	if err != nil {
		return nil, jwt.ErrTokenSignatureInvalid
	}
	claims, ok := token.Claims.(*claims)
	if !ok {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}

func JwtMiddleware(queries *repositry.Queries) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tokenString := strings.TrimPrefix(c.Get(fiber.HeaderAuthorization), "Bearer ")
		if tokenString == "" {
			return fiber.NewError(fiber.StatusBadRequest, "missing or malformed Authorization header")
		}
		claims, err := parseJWTTokenString(tokenString)
		if err != nil {
			return fiber.ErrUnauthorized
		}
		if claims.ExpiresAt.Sub(time.Now()) < 0 {
			return fiber.ErrUnauthorized
		}
		userID := claims.UserID
		// NOTE: if the users deleted his account, but his access token hasn't expired yet,
		// and we got a request that uses mwAuth(get's userid from context),
		// we need to ensure that user exists.
		if exists, err := queries.CheckUserID(context.Background(), userID); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking user ID: %+v", err))
		} else if !exists {
			return fiber.ErrUnauthorized
		}
		c.Locals("userID", userID)
		return c.Next()
	}
}
