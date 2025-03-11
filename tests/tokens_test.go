package utils

import (
	"os"
	"testing"
	"time"

	"github.com/assaidy/blogging_app/internal/utils"
	"github.com/stretchr/testify/assert"
)

// TODO: move to tests package

func TestGenerateRefreshToken(t *testing.T) {
	os.Setenv("REFRESH_TOKEN_EXPIRATION_DAYS", "7")
	defer os.Unsetenv("REFRESH_TOKEN_EXPIRATION_DAYS")

	token, err := utils.GenerateRefreshToken()
	assert.NoError(t, err)
	assert.NotEmpty(t, token.Token)
	assert.True(t, token.ExpiresAt.After(time.Now()))
}

func TestGenerateJWTAccessToken(t *testing.T) {
	os.Setenv("ACCESS_TOKEN_EXPIRATION_MINUTES", "30")
	os.Setenv("SECRET", "test-secret")
	defer os.Unsetenv("ACCESS_TOKEN_EXPIRATION_MINUTES")
	defer os.Unsetenv("SECRET")

	userID := "test-user-id"
	tokenString, err := utils.GenerateJWTAccessToken(userID)
	assert.NoError(t, err)
	assert.NotEmpty(t, tokenString)

	claims, err := utils.ParseJWTTokenString(tokenString)
	assert.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.True(t, claims.ExpiresAt.Time.After(time.Now()))
}

func TestParseJWTTokenString(t *testing.T) {
	os.Setenv("ACCESS_TOKEN_EXPIRATION_MINUTES", "30")
	os.Setenv("SECRET", "test-secret")
	defer os.Unsetenv("ACCESS_TOKEN_EXPIRATION_MINUTES")
	defer os.Unsetenv("SECRET")

	userID := "test-user-id"
	tokenString, err := utils.GenerateJWTAccessToken(userID)
	assert.NoError(t, err)

	claims, err := utils.ParseJWTTokenString(tokenString)
	assert.Equal(t, userID, claims.UserID)

	_, err = utils.ParseJWTTokenString("invalid-token")
	assert.Error(t, err)
}
