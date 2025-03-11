package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/assaidy/blogging_app/internal/repo"
	"github.com/assaidy/blogging_app/internal/repo/postgres_repo"
	"github.com/assaidy/blogging_app/internal/types"
	"github.com/assaidy/blogging_app/internal/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/oklog/ulid/v2"
)

func HandleRegister(c *fiber.Ctx) error {
	req := types.UserRegisterRequest{}
	if err := parseAndValidateJsonBody(c, &req); err != nil {
		return err
	}

	if exists, err := queries.CheckUsername(context.Background(), req.Username); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking username: %+v", err))
	} else if exists {
		return fiber.NewError(fiber.StatusConflict, "username already exists")
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error hashing password: %+v", err))
	}

	user, err := queries.CreateUser(context.Background(), postgres_repo.CreateUserParams{
		ID:             ulid.Make().String(),
		Name:           req.Name,
		Username:       req.Username,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error storing user: %+v", err))
	}

	accessToken, err := utils.GenerateJWTAccessToken(user.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error creating access token: %+v", err))
	}
	refreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error creating refresh token: %+v", err))
	}

	if err := queries.CreateRefreshToken(context.Background(), postgres_repo.CreateRefreshTokenParams{
		Token:     refreshToken.Token,
		UserID:    user.ID,
		ExpiresAt: refreshToken.ExpiresAt,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error storing refresh token: %+v", err))
	}

	var userPayload types.UserPayload
	fillUserPayload(&userPayload, &user)

	return c.Status(fiber.StatusCreated).JSON(types.ApiResponse{
		Payload:      userPayload,
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
	})
}

func HandleLogin(c *fiber.Ctx) error {
	req := types.UserLoginRequest{}
	if err := parseAndValidateJsonBody(c, &req); err != nil {
		return err
	}

	user, err := queries.GetUserByUsername(context.Background(), req.Username)
	if err != nil {
		if repo.IsNotFoundError(err) {
			return fiber.ErrUnauthorized
		}
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting user: %+v", err))
	}

	if !utils.VerifyPassword(req.Password, user.HashedPassword) {
		return fiber.ErrUnauthorized
	}

	accessToken, err := utils.GenerateJWTAccessToken(user.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error creating access token: %+v", err))
	}
	refreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error creating refresh token: %+v", err))
	}

	if err := queries.CreateRefreshToken(context.Background(), postgres_repo.CreateRefreshTokenParams{
		Token:     refreshToken.Token,
		UserID:    user.ID,
		ExpiresAt: refreshToken.ExpiresAt,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error storing refresh token: %+v", err))
	}

	var userPayload types.UserPayload
	fillUserPayload(&userPayload, &user)

	return c.Status(fiber.StatusCreated).JSON(types.ApiResponse{
		Payload:      userPayload,
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
	})
}

func HandleGetAccessToken(c *fiber.Ctx) error {
	refreshTokenQuery := c.Query("refreshToken")

	refreshToken, err := queries.GetRefreshToken(context.Background(), refreshTokenQuery)
	if err != nil {
		if repo.IsNotFoundError(err) {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
		}
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting refresh token: %+v", err))
	}

	if refreshToken.ExpiresAt.Sub(time.Now()) < 0 {
		if err := queries.DeleteRefreshToken(context.Background(), refreshToken.Token); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error deleting refresh token: %+v", err))
		}
		return fiber.NewError(fiber.StatusUnprocessableEntity, fmt.Sprintf("refresh token expired: %+v", err))
	}

	accessToken, err := utils.GenerateJWTAccessToken(refreshToken.UserID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error creating access token: %+v", err))
	}

	return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
		AccessToken: accessToken,
	})
}
