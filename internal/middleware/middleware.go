package middleware

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/assaidy/blogging_app/internal/db/postgres_db"
	"github.com/assaidy/blogging_app/internal/repo/postgres_repo"
	"github.com/assaidy/blogging_app/internal/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

var repo = postgres_repo.New(postgres_db.DB)

const AuthUserID = "middleware.auth.userID"

var Logger = logger.New()

func Auth(c *fiber.Ctx) error {
	tokenString := strings.TrimPrefix(c.Get(fiber.HeaderAuthorization), "Bearer ")
	if tokenString == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing or malformed Authorization header")
	}
	claims, err := utils.ParseJWTTokenString(tokenString)
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
	if exists, err := repo.CheckUserID(context.Background(), userID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking user ID: %+v", err))
	} else if !exists {
		return fiber.ErrUnauthorized
	}
	c.Locals(AuthUserID, userID)
	return c.Next()
}

func ErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		code = fiberErr.Code
	}
	c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
	// NOTE: Logging occurs before this error handler is executed, so the internal error
	// has already been logged. We avoid exposing internal error details to the client
	// by returning a generic error mfessage.
	if code == fiber.StatusInternalServerError {
		return c.SendStatus(code)
	}
	return c.Status(code).SendString(err.Error())
}
