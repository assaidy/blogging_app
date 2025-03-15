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

func HandleGetAllUsers(c *fiber.Ctx) error {
	limit := c.QueryInt("limit")
	if limit < 10 || limit > 100 {
		limit = 10
	}

	var requestCursor usersCursor
	if err := decodeBase64AndUnmarshalJson(&requestCursor, c.Query("cursor")); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid cursor format")
	}

	users, err := queries.GetAllUsers(context.Background(), postgres_repo.GetAllUsersParams{
		// == cursor
		Followerscount: int32(requestCursor.FollowersCount),
		Postscount:     int32(requestCursor.PostsCount),
		ID:             requestCursor.ID,
		// == filter
		Name:     c.Query("name"),
		Username: c.Query("username"),
		// == limit
		Limit: int32(limit + 1),
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting users: %+v", err))
	}

	var encodedResponseCursor string
	hasMore := limit < len(users)
	if hasMore {
		responseCursor := usersCursor{
			FollowersCount: int(users[limit].FollowersCount),
			PostsCount:     int(users[limit].PostsCount),
			ID:             users[limit].ID,
		}
		encodedResponseCursor, err = marshalJsonAndEncodeBase64(responseCursor)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error encoding cursor: %+v", err))
		}
		users = users[:limit]
	}

	payload := make([]types.UserPayload, 0, len(users))
	for _, user := range users {
		var userPayload types.UserPayload
		fillUserPayload(&userPayload, &user)
		payload = append(payload, userPayload)
	}

	return c.Status(fiber.StatusOK).JSON(cursoredApiResponse{
		Payload:    payload,
		Cursor:     encodedResponseCursor,
		HasMore:    hasMore,
		TotalCount: len(payload),
	})
}

func HandleGetAllFollowers(c *fiber.Ctx) error {
	limit := c.QueryInt("limit")
	if limit < 10 || limit > 100 {
		limit = 10
	}

	var requestCursor followersCursor
	if err := decodeBase64AndUnmarshalJson(&requestCursor, c.Query("cursor")); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid cursor format")
	}

	followers, err := queries.GetAllFollowers(context.Background(), postgres_repo.GetAllFollowersParams{
		// == filter
		FollowedID: getUserIDFromContext(c),
		// == cursor
		ID: requestCursor.ID,
		// == limit
		Limit: int32(limit + 1),
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting followers: %+v", err))
	}

	var encodedResponseCursor string
	hasMore := limit < len(followers)
	if hasMore {
		responseCursor := followersCursor{
			ID: followers[limit].ID,
		}
		encodedResponseCursor, err = marshalJsonAndEncodeBase64(responseCursor)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error encoding cursor: %+v", err))
		}
		followers = followers[:limit]
	}

	payload := make([]types.UserPayload, 0, len(followers))
	for _, follower := range followers {
		var userPayload types.UserPayload
		fillUserPayload(&userPayload, &follower)
		payload = append(payload, userPayload)
	}

	return c.Status(fiber.StatusOK).JSON(cursoredApiResponse{
		Payload:    payload,
		Cursor:     encodedResponseCursor,
		HasMore:    hasMore,
		TotalCount: len(payload),
	})
}
