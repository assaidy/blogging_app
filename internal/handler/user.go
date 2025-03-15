package handler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/assaidy/blogging_app/internal/repo"
	"github.com/assaidy/blogging_app/internal/repo/postgres_repo"
	"github.com/assaidy/blogging_app/internal/types"
	"github.com/assaidy/blogging_app/internal/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/oklog/ulid/v2"
)

func HandleGetUserById(c *fiber.Ctx) error {
	userID := c.Params("user_id")
	if !utils.IsValidEncodedULID(userID) {
		return fiber.NewError(fiber.StatusBadRequest, "invalid ID format")
	}

	user, err := queries.GetUserByID(context.Background(), userID)
	if err != nil {
		if repo.IsNotFoundError(err) {
			return fiber.NewError(fiber.StatusNotFound, "user not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting user: %+v", err))
	}

	var userPayload types.UserPayload
	fillUserPayload(&userPayload, &user)

	return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
		Payload: userPayload,
	})
}

func HandleGetUserByUsername(c *fiber.Ctx) error {
	username := c.Params("username")

	user, err := queries.GetUserByUsername(context.Background(), username)
	if err != nil {
		if repo.IsNotFoundError(err) {
			return fiber.NewError(fiber.StatusNotFound, "user not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting user: %+v", err))
	}

	var userPayload types.UserPayload
	fillUserPayload(&userPayload, &user)

	return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
		Payload: userPayload,
	})
}

func HandleUpdateUser(c *fiber.Ctx) error {
	req := types.UserUpdateRequest{}
	if err := parseAndValidateJsonBody(c, &req); err != nil {
		return err
	}

	userID := getUserIDFromContext(c)

	oldUser, err := queries.GetUserByID(context.Background(), userID)
	if err != nil {
		if repo.IsNotFoundError(err) {
			return fiber.NewError(fiber.StatusNotFound, "user not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting user: %+v", err))
	}

	if oldUser.Username != req.Username {
		if exists, err := queries.CheckUsername(context.Background(), req.Username); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking username: %+v", err))
		} else if exists {
			return fiber.NewError(fiber.StatusConflict, "username already exists")
		}
	}

	if !utils.VerifyPassword(req.OldPassword, oldUser.HashedPassword) {
		return fiber.NewError(fiber.StatusForbidden, "invalid old password")
	}

	newHashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error hashing password: %+v", err))
	}

	newUser, err := queries.UpdateUser(context.Background(), postgres_repo.UpdateUserParams{
		ID:              userID,
		Name:            req.Name,
		Username:        req.Username,
		HashedPassword:  newHashedPassword,
		ProfileImageUrl: sql.NullString{Valid: true, String: req.ProfileImageUrl},
	})

	var userPayload types.UserPayload
	fillUserPayload(&userPayload, &newUser)

	return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
		Payload: userPayload,
	})
}

func HandleDeleteUser(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)

	if err := queries.DeleteUser(context.Background(), userID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error deleting user: %+v", err))
	}

	return c.Status(fiber.StatusOK).SendString("user deleted successfully")
}

func HandleFollow(c *fiber.Ctx) error {
	followedID := c.Params("followed_id")
	if !utils.IsValidEncodedULID(followedID) {
		return fiber.NewError(fiber.StatusBadRequest, "invalid ID format")
	}
	userID := getUserIDFromContext(c)

	if userID == followedID {
		return fiber.NewError(fiber.StatusForbidden, "user can't unfollow himself")
	}

	if exists, err := queries.CheckUserID(context.Background(), followedID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking user ID: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}

	if exists, err := queries.CheckFollow(context.Background(), postgres_repo.CheckFollowParams{
		FollowerID: userID,
		FollowedID: followedID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking follow: %+v", err))
	} else if exists {
		return fiber.NewError(fiber.StatusConflict, "user is already followed")
	}

	if err := queries.CreateFollow(context.Background(), postgres_repo.CreateFollowParams{
		FollowerID: userID,
		FollowedID: followedID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error creating follow: %+v", err))
	}

	notificationChan <- postgres_repo.Notification{
		ID:       ulid.Make().String(),
		KindID:   repo.NotificationKindNewFollower,
		UserID:   followedID,
		SenderID: userID,
	}

	return c.Status(fiber.StatusOK).SendString("user was followed successfully")
}

func HandleUnfollow(c *fiber.Ctx) error {
	followedID := c.Params("followed_id")
	if !utils.IsValidEncodedULID(followedID) {
		return fiber.NewError(fiber.StatusBadRequest, "invalid ID fromat")
	}
	userID := getUserIDFromContext(c)

	if userID == followedID {
		return fiber.NewError(fiber.StatusForbidden, "user can't follow himself")
	}

	if exists, err := queries.CheckUserID(context.Background(), followedID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking user ID: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}

	if exists, err := queries.CheckFollow(context.Background(), postgres_repo.CheckFollowParams{
		FollowerID: userID,
		FollowedID: followedID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking follow: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "follow not found")
	}

	if err := queries.DeleteFollow(context.Background(), postgres_repo.DeleteFollowParams{
		FollowerID: userID,
		FollowedID: followedID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error deleting follow: %+v", err))
	}

	return c.Status(fiber.StatusOK).SendString("user was unfollowed successfully")
}
