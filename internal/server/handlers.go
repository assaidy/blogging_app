package server

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/assaidy/blogging_app/internal/repositry"
	"github.com/assaidy/blogging_app/internal/types"
	"github.com/assaidy/blogging_app/internal/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/oklog/ulid/v2"
)

// TODO:
// - use cursor pagination
// - remove this const and get the limit as a query parm
const pageSize = 20

func (me *App) handleRegister(c *fiber.Ctx) error {
	req := types.UserRegisterRequest{}
	if err := parseAndValidateJsonBody(c, &req); err != nil {
		return err
	}

	if exists, err := me.queries.CheckUsername(context.Background(), req.Username); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking username: %+v", err))
	} else if exists {
		return fiber.NewError(fiber.StatusConflict, "username already exists")
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error hashing password: %+v", err))
	}

	user, err := me.queries.CreateUser(context.Background(), repositry.CreateUserParams{
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
	refreshToken, expiresAt, err := utils.GenerateRefreshToken()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error creating refresh token: %+v", err))
	}

	if err := me.queries.CreateRefreshToken(context.Background(), repositry.CreateRefreshTokenParams{
		Token:     refreshToken,
		UserID:    user.ID,
		ExpiresAt: expiresAt,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error storing refresh token: %+v", err))
	}

	var userPayload types.UserPayload
	fillUserPayload(&userPayload, &user)

	return c.Status(fiber.StatusCreated).JSON(types.ApiResponse{
		Payload:      userPayload,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

func (me *App) handleLogin(c *fiber.Ctx) error {
	req := types.UserLoginRequest{}
	if err := parseAndValidateJsonBody(c, &req); err != nil {
		return err
	}

	user, err := me.queries.GetUserByUsername(context.Background(), req.Username)
	if err != nil {
		if repositry.IsNotFoundError(err) {
			return fiber.ErrUnauthorized
		}
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting user: %+v", err))
	}

	if !utils.VerfityPassword(req.Password, user.HashedPassword) {
		return fiber.ErrUnauthorized
	}

	accessToken, err := utils.GenerateJWTAccessToken(user.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error creating access token: %+v", err))
	}
	refreshToken, expiresAt, err := utils.GenerateRefreshToken()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error creating refresh token: %+v", err))
	}

	if err := me.queries.CreateRefreshToken(context.Background(), repositry.CreateRefreshTokenParams{
		Token:     refreshToken,
		UserID:    user.ID,
		ExpiresAt: expiresAt,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error storing refresh token: %+v", err))
	}

	var userPayload types.UserPayload
	fillUserPayload(&userPayload, &user)

	return c.Status(fiber.StatusCreated).JSON(types.ApiResponse{
		Payload:      userPayload,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

func (me *App) handleGetAccessToken(c *fiber.Ctx) error {
	refreshTokenQuery := c.Query("refreshToken")

	refreshToken, err := me.queries.GetRefreshToken(context.Background(), refreshTokenQuery)
	if err != nil {
		if repositry.IsNotFoundError(err) {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
		}
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting refresh token: %+v", err))
	}

	if refreshToken.ExpiresAt.Sub(time.Now()) < 0 {
		if err := me.queries.DeleteRefreshToken(context.Background(), refreshToken.Token); err != nil {
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

func (me *App) handleGetUserById(c *fiber.Ctx) error {
	userID := c.Params("user_id")

	user, err := me.queries.GetUserByID(context.Background(), userID)
	if err != nil {
		if repositry.IsNotFoundError(err) {
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

func (me *App) handleGetUserByUsername(c *fiber.Ctx) error {
	username := c.Params("username")

	user, err := me.queries.GetUserByUsername(context.Background(), username)
	if err != nil {
		if repositry.IsNotFoundError(err) {
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

func (me *App) handleUpdateUser(c *fiber.Ctx) error {
	req := types.UserUpdateRequest{}
	if err := parseAndValidateJsonBody(c, &req); err != nil {
		return err
	}

	userID := getUserIDFromContext(c)

	oldUser, err := me.queries.GetUserByID(context.Background(), userID)
	if err != nil {
		if repositry.IsNotFoundError(err) {
			return fiber.NewError(fiber.StatusNotFound, "user not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting user: %+v", err))
	}

	if oldUser.Username != req.Username {
		if exists, err := me.queries.CheckUsername(context.Background(), req.Username); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking username: %+v", err))
		} else if exists {
			return fiber.NewError(fiber.StatusConflict, "username already exists")
		}
	}

	if !utils.VerfityPassword(req.OldPassword, oldUser.HashedPassword) {
		return fiber.NewError(fiber.StatusForbidden, "invalid old password")
	}

	newHashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error hashing password: %+v", err))
	}

	newUser, err := me.queries.UpdateUser(context.Background(), repositry.UpdateUserParams{
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

func (me *App) handleDeleteUser(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)

	if err := me.queries.DeleteUser(context.Background(), userID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error deleting user: %+v", err))
	}

	return c.Status(fiber.StatusOK).SendString("user deleted successfully")
}

func (me *App) handleFollow(c *fiber.Ctx) error {
	followedID := c.Params("followed_id")
	userID := getUserIDFromContext(c)

	if userID == followedID {
		return fiber.NewError(fiber.StatusForbidden, "user can't unfollow himself")
	}

	if exists, err := me.queries.CheckUserID(context.Background(), followedID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking user ID: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}

	if exists, err := me.queries.CheckFollow(context.Background(), repositry.CheckFollowParams{
		FollowerID: userID,
		FollowedID: followedID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking follow: %+v", err))
	} else if exists {
		return fiber.NewError(fiber.StatusConflict, "user is already followed")
	}

	if err := me.queries.CreateFollow(context.Background(), repositry.CreateFollowParams{
		FollowerID: userID,
		FollowedID: followedID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error creating follow: %+v", err))
	}

	me.notificationChan <- repositry.Notification{
		ID:       ulid.Make().String(),
		KindID:   repositry.NotificationKindNewFollower,
		UserID:   followedID,
		SenderID: sql.NullString{Valid: true, String: userID},
	}

	return c.Status(fiber.StatusOK).SendString("user was followed successfully")
}

func (me *App) handleUnfollow(c *fiber.Ctx) error {
	followedID := c.Params("followed_id")
	userID := getUserIDFromContext(c)

	if userID == followedID {
		return fiber.NewError(fiber.StatusForbidden, "user can't follow himself")
	}

	if exists, err := me.queries.CheckUserID(context.Background(), followedID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking user ID: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}

	if exists, err := me.queries.CheckFollow(context.Background(), repositry.CheckFollowParams{
		FollowerID: userID,
		FollowedID: followedID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking follow: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "follow not found")
	}

	if err := me.queries.DeleteFollow(context.Background(), repositry.DeleteFollowParams{
		FollowerID: userID,
		FollowedID: followedID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error deleting follow: %+v", err))
	}

	return c.Status(fiber.StatusOK).SendString("user was unfollowed successfully")
}

// func (me *App) handleGetAllFollowers(c *fiber.Ctx) error {
// 	userID := getUserIDFromContext(c)
// 	limit := c.QueryInt("limit")
// 	// encodedCursor := c.Query("cursor")
// 	// return {payload: any, cursor: base64 string, hasMore: bool}

// 	followers, err := me.queries.GetFollowers(context.Background(), repositry.GetFollowersParams{
// 		FollowedID: userID,
// 		Limit:      pageSize,
// 	})
// 	if err != nil {
// 		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting followers: %+v", err))
// 	}

// 	payload := make([]types.UserPayload, 0, len(followers))
// 	for _, follower := range followers {
// 		payload = append(payload, types.UserPayload{
// 			ID:              follower.ID,
// 			Name:            follower.Name,
// 			Username:        follower.Username,
// 			JoinedAt:        follower.JoinedAt,
// 			PostsCount:      follower.PostsCount,
// 			FollowingCount:  follower.FollowingCount,
// 			FollowersCount:  follower.FollowersCount,
// 			ProfileImageUrl: follower.ProfileImageUrl.String,
// 		})
// 	}

// 	return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
// 		Payload: payload,
// 	})
// }

func (me *App) handleCreatePost(c *fiber.Ctx) error {
	req := types.PostCreateOrUpdateRequest{}
	if err := parseAndValidateJsonBody(c, &req); err != nil {
		return err
	}

	userID := getUserIDFromContext(c)

	post, err := me.queries.CreatePost(context.Background(), repositry.CreatePostParams{
		ID:               ulid.Make().String(),
		UserID:           userID,
		Title:            req.Title,
		Content:          req.Content,
		FeaturedImageUrl: sql.NullString{Valid: true, String: req.FeaturedImageUrl},
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error creating post: %+v", err))
	}

	followersIDs, err := me.queries.GetAllFollowersIDs(context.Background(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting followers IDs: %+v", err))
	}
	for _, id := range followersIDs {
		me.notificationChan <- repositry.Notification{
			ID:       ulid.Make().String(),
			KindID:   repositry.NotificationKindNewPost,
			UserID:   id,
			SenderID: sql.NullString{Valid: true, String: post.UserID},
			PostID:   sql.NullString{Valid: true, String: post.ID},
		}
	}

	var payload types.PostPayload
	fillPostPayload(&payload, &post)
	payload.Reactions = []types.PostPayloadReaction{}

	return c.Status(fiber.StatusCreated).JSON(types.ApiResponse{
		Payload: payload,
	})
}

func (me *App) handleGetPost(c *fiber.Ctx) error {
	postID := c.Params("post_id")

	post, err := me.queries.GetPost(context.Background(), postID)
	if err != nil {
		if repositry.IsNotFoundError(err) {
			return fiber.NewError(fiber.StatusNotFound, "post not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting post: %+v", err))
	}

	reactions, err := me.queries.GetPostReactions(context.Background(), postID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting post reactions: %+v", err))
	}

	var payload types.PostPayload
	fillPostPayload(&payload, &post)
	fillPostReactions(&payload, reactions)

	return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
		Payload: payload,
	})
}

func (me *App) handleUpdatePost(c *fiber.Ctx) error {
	req := types.PostCreateOrUpdateRequest{}
	if err := parseAndValidateJsonBody(c, &req); err != nil {
		return err
	}

	postID := c.Params("post_id")

	if exists, err := me.queries.CheckPost(context.Background(), postID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking post: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "post not found")
	}

	userID := getUserIDFromContext(c)

	if ok, err := me.queries.CheckUserOwnsPost(context.Background(), repositry.CheckUserOwnsPostParams{
		ID:     postID,
		UserID: userID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking user owns post: %+v", err))
	} else if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "you don't own this post")
	}

	newPost, err := me.queries.UpdatePost(context.Background(), repositry.UpdatePostParams{
		ID:               postID,
		Title:            req.Title,
		Content:          req.Content,
		FeaturedImageUrl: sql.NullString{Valid: true, String: req.FeaturedImageUrl},
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error updating post: %+v", err))
	}

	reactions, err := me.queries.GetPostReactions(context.Background(), postID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting post reactions: %+v", err))
	}

	var payload types.PostPayload
	fillPostPayload(&payload, &newPost)
	fillPostReactions(&payload, reactions)

	return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
		Payload: payload,
	})
}

func (me *App) handleDeletePost(c *fiber.Ctx) error {
	postID := c.Params("post_id")

	if exists, err := me.queries.CheckPost(context.Background(), postID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking post: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "post not found")
	}

	userID := getUserIDFromContext(c)

	if ok, err := me.queries.CheckUserOwnsPost(context.Background(), repositry.CheckUserOwnsPostParams{
		ID:     postID,
		UserID: userID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking user owns post: %+v", err))
	} else if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "you don't own this post")
	}

	if err := me.queries.DeletePost(context.Background(), postID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error deleting post: %+v", err))
	}

	return c.Status(fiber.StatusOK).SendString("post deleted successfully")
}

func (me *App) handleGetAllUserPosts(c *fiber.Ctx) error {
	userID := c.Params("user_id")

	if exists, err := me.queries.CheckUserID(context.Background(), userID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking user ID: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}

	page := c.QueryInt("page")
	if page < 1 {
		page = 1
	}

	postsCount, err := me.queries.GetUserPostsCount(context.Background(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting user posts count: %+v", err))
	}

	totalPages := math.Ceil(float64(postsCount) / pageSize)
	if page > int(totalPages) {
		return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
			Payload:    []any{},
			TotalPages: int(totalPages),
		})
	}

	offset := (page - 1) * pageSize
	posts, err := me.queries.GetUserPosts(context.Background(), repositry.GetUserPostsParams{
		UserID: userID,
		Limit:  pageSize,
		Offset: int32(offset),
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting user posts: %+v", err))
	}

	payload := make([]types.PostPayload, 0, len(posts))
	for _, post := range posts {
		curr := types.PostPayload{
			ID:               post.ID,
			UserID:           post.UserID,
			Title:            post.Title,
			Content:          post.Content,
			CreatedAt:        post.CreatedAt,
			ViewsCount:       post.ViewsCount,
			CommentsCount:    post.CommentsCount,
			FeaturedImageUrl: post.FeaturedImageUrl.String,
		}
		reactions, err := me.queries.GetPostReactions(context.Background(), post.ID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting post reactions: %+v", err))
		}
		for _, reaction := range reactions {
			curr.Reactions = append(curr.Reactions, types.PostPayloadReaction{
				Name:  reaction.Name,
				Count: reaction.Count,
			})
		}
		payload = append(payload, curr)
	}

	return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
		Payload:    payload,
		TotalPages: int(totalPages),
	})
}

func (me *App) handleViewPost(c *fiber.Ctx) error {
	postID := c.Params("post_id")

	if exists, err := me.queries.CheckPost(context.Background(), postID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking post: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "post not found")
	}

	userID := getUserIDFromContext(c)

	if ok, err := me.queries.CheckUserOwnsPost(context.Background(), repositry.CheckUserOwnsPostParams{
		ID:     postID,
		UserID: userID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking user owns post: %+v", err))
	} else if ok {
		return fiber.NewError(fiber.StatusForbidden, "we don't count user viewing his own post")
	}

	if err := me.queries.ViewPost(context.Background(), repositry.ViewPostParams{
		PostID: postID,
		UserID: userID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error viewing a post: %+v", err))
	}

	return c.Status(fiber.StatusOK).SendString("post view was added successfully")
}

func (me *App) handleGetAllPostViews(c *fiber.Ctx) error {
	postID := c.Params("post_id")

	if exists, err := me.queries.CheckPost(context.Background(), postID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking post: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "post not found")
	}

	userID := getUserIDFromContext(c)

	if ok, err := me.queries.CheckUserOwnsPost(context.Background(), repositry.CheckUserOwnsPostParams{
		ID:     postID,
		UserID: userID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking user owns post: %+v", err))
	} else if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "you don't own this post")
	}

	page := c.QueryInt("page")
	if page < 1 {
		page = 1
	}

	viewsCount, err := me.queries.GetPostViewsCount(context.Background(), postID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting post views count: %+v", err))
	}

	totalPages := math.Ceil(float64(viewsCount) / pageSize)
	if page > int(totalPages) {
		return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
			Payload:    []any{},
			TotalPages: int(totalPages),
		})
	}

	offset := (page - 1) * pageSize
	users, err := me.queries.GetPostViews(context.Background(), repositry.GetPostViewsParams{
		PostID: postID,
		Limit:  pageSize,
		Offset: int32(offset),
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting post views: %+v", err))
	}

	payload := make([]types.UserPayload, 0, len(users))
	for _, user := range users {
		var userPayload types.UserPayload
		fillUserPayload(&userPayload, &user)
		payload = append(payload, userPayload)
	}

	return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
		Payload:    payload,
		TotalPages: int(totalPages),
	})
}

func (me *App) handleCreateComment(c *fiber.Ctx) error {
	req := types.CommentCreateOrUpdateRequest{}
	if err := parseAndValidateJsonBody(c, &req); err != nil {
		return err
	}

	postID := c.Params("post_id")

	if exists, err := me.queries.CheckPost(context.Background(), postID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking post: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "post not found")
	}

	userID := getUserIDFromContext(c)

	comment, err := me.queries.CreateComment(context.Background(), repositry.CreateCommentParams{
		ID:      ulid.Make().String(),
		PostID:  postID,
		UserID:  userID,
		Content: req.Content,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error creating error: %+v", err))
	}

	var payload types.CommentPayload
	fillCommentPayload(&payload, &comment)

	return c.Status(fiber.StatusCreated).JSON(types.ApiResponse{
		Payload: payload,
	})
}

func (me *App) handleUpdateComment(c *fiber.Ctx) error {
	req := types.CommentCreateOrUpdateRequest{}
	if err := parseAndValidateJsonBody(c, &req); err != nil {
		return err
	}

	commentID := c.Params("comment_id")

	if exists, err := me.queries.CheckComment(context.Background(), commentID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking comment: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "comment not found")
	}

	userID := getUserIDFromContext(c)

	if ok, err := me.queries.CheckUserOwnsComment(context.Background(), repositry.CheckUserOwnsCommentParams{
		ID:     commentID,
		UserID: userID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking user owns comment: %+v", err))
	} else if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "you don't own this comment")
	}

	newComment, err := me.queries.UpdateComment(context.Background(), repositry.UpdateCommentParams{
		ID:      commentID,
		Content: req.Content,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error updating comment: %+v", err))
	}

	var payload types.CommentPayload
	fillCommentPayload(&payload, &newComment)

	return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
		Payload: payload,
	})
}

func (me *App) handleDeleteComment(c *fiber.Ctx) error {
	commentID := c.Params("comment_id")

	if exists, err := me.queries.CheckComment(context.Background(), commentID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking comment: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "comment not found")
	}

	userID := getUserIDFromContext(c)

	if ok, err := me.queries.CheckUserOwnsComment(context.Background(), repositry.CheckUserOwnsCommentParams{
		ID:     commentID,
		UserID: userID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking user owns comment: %+v", err))
	} else if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "you don't own this comment")
	}

	if err := me.queries.DeleteComment(context.Background(), commentID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error deleting post: %+v", err))
	}

	return c.Status(fiber.StatusOK).SendString("comment deleted successfully")
}

func (me *App) handleGetAllPostComments(c *fiber.Ctx) error {
	postID := c.Params("post_id")

	if exists, err := me.queries.CheckPost(context.Background(), postID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking post: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "post not found")
	}

	page := c.QueryInt("page")
	if page < 1 {
		page = 1
	}

	commentsCount, err := me.queries.GetPostCommentsCount(context.Background(), postID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting comments count: %+v", err))
	}

	totalPages := math.Ceil(float64(commentsCount) / pageSize)
	if page > int(totalPages) {
		return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
			Payload:    []any{},
			TotalPages: int(totalPages),
		})
	}

	offset := (page - 1) * pageSize
	comments, err := me.queries.GetPostComments(context.Background(), repositry.GetPostCommentsParams{
		PostID: postID,
		Limit:  pageSize,
		Offset: int32(offset),
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting comments: %+v", err))
	}

	payload := make([]types.CommentPayload, 0, len(comments))
	for _, comment := range comments {
		payload = append(payload, types.CommentPayload{
			ID:        comment.ID,
			PostID:    comment.PostID,
			UserID:    comment.UserID,
			Content:   comment.Content,
			CreatedAt: comment.CreatedAt,
		})
	}

	return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
		Payload:    payload,
		TotalPages: int(totalPages),
	})
}

func (me *App) handleAddToBookmarks(c *fiber.Ctx) error {
	postID := c.Params("post_id")

	if exists, err := me.queries.CheckPost(context.Background(), postID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking post: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "post not found")
	}

	userID := getUserIDFromContext(c)

	if exists, err := me.queries.CheckBookmark(context.Background(), repositry.CheckBookmarkParams{
		PostID: postID,
		UserID: userID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking bookmark: %+v", err))
	} else if exists {
		return fiber.NewError(fiber.StatusConflict, "bookmarks already exists")
	}

	if err := me.queries.CreateBookmark(context.Background(), repositry.CreateBookmarkParams{
		PostID: postID,
		UserID: userID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error creating bookmark: %+v", err))
	}

	return c.Status(fiber.StatusCreated).SendString("bookmark created successfully")
}

func (me *App) handleDeleteFromBookmarks(c *fiber.Ctx) error {
	postID := c.Params("post_id")
	userID := getUserIDFromContext(c)

	if exists, err := me.queries.CheckBookmark(context.Background(), repositry.CheckBookmarkParams{
		PostID: postID,
		UserID: userID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking bookmark: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "bookmark doesn't exist")
	}

	if err := me.queries.DeleteBookmark(context.Background(), repositry.DeleteBookmarkParams{
		PostID: postID,
		UserID: userID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error deleting bookmark: %+v", err))
	}

	return c.Status(fiber.StatusOK).SendString("bookmark deleted successfully")
}

func (me *App) handleGetAllBookmarks(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)

	bookmarksCount, err := me.queries.GetBookmarksCount(context.Background(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting bookmarks count: %+v", err))
	}

	page := c.QueryInt("page")
	if page < 1 {
		page = 1
	}

	totalPages := math.Ceil(float64(bookmarksCount) / pageSize)
	if page > int(totalPages) {
		return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
			Payload:    []any{},
			TotalPages: int(totalPages),
		})
	}

	offset := (page - 1) * pageSize
	bookmarks, err := me.queries.GetBookmarks(context.Background(), repositry.GetBookmarksParams{
		UserID: userID,
		Limit:  pageSize,
		Offset: int32(offset),
	})

	payload := make([]types.PostPayload, 0, len(bookmarks))
	for _, bookmark := range bookmarks {
		curr := types.PostPayload{
			ID:               bookmark.ID,
			UserID:           bookmark.UserID,
			Title:            bookmark.Title,
			Content:          bookmark.Content,
			CreatedAt:        bookmark.CreatedAt,
			ViewsCount:       bookmark.ViewsCount,
			CommentsCount:    bookmark.CommentsCount,
			FeaturedImageUrl: bookmark.FeaturedImageUrl.String,
		}
		reactions, err := me.queries.GetPostReactions(context.Background(), bookmark.ID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting post reactions: %+v", err))
		}
		for _, reaction := range reactions {
			curr.Reactions = append(curr.Reactions, types.PostPayloadReaction{
				Name:  reaction.Name,
				Count: reaction.Count,
			})
		}
		payload = append(payload, curr)
	}

	return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
		Payload:    payload,
		TotalPages: int(totalPages),
	})
}

func (me *App) handleReact(c *fiber.Ctx) error {
	postID := c.Params("post_id")
	reactionKindName := c.Query("reaction_kind")

	if exists, err := me.queries.CheckPost(context.Background(), postID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking post: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "post not found")
	}

	kindID, err := me.queries.GetReactionKindIDByName(context.Background(), reactionKindName)
	if err != nil {
		if repositry.IsNotFoundError(err) {
			return fiber.NewError(fiber.StatusBadRequest, "invalid reaction kind")
		}
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting reaction kind id: %+v", err))
	}

	userID := getUserIDFromContext(c)

	if err := me.queries.CreateReaction(context.Background(), repositry.CreateReactionParams{
		PostID: postID,
		UserID: userID,
		KindID: kindID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error creating reaction: %+v", err))
	}

	return c.Status(fiber.StatusCreated).SendString("reaction added successfully")
}

func (me *App) handleDeleteReaction(c *fiber.Ctx) error {
	postID := c.Params("post_id")
	userID := getUserIDFromContext(c)

	if exists, err := me.queries.CheckReaction(context.Background(), repositry.CheckReactionParams{
		PostID: postID,
		UserID: userID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking reaction: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "you have no reactions on this post")
	}

	if err := me.queries.DeleteReaction(context.Background(), repositry.DeleteReactionParams{
		PostID: postID,
		UserID: userID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error deleting reaction: %+v", err))
	}

	return c.Status(fiber.StatusOK).SendString("reaction deleted successfully")
}

func (me *App) StartNotificationWorker(ctx context.Context) {
	for {
		select {
		case notification := <-me.notificationChan:
			fmt.Println("recieved notification")
			if _, err := me.queries.CreateNotification(context.Background(), repositry.CreateNotificationParams{
				ID:       notification.ID,
				KindID:   notification.KindID,
				UserID:   notification.UserID,
				SenderID: notification.SenderID,
				PostID:   notification.PostID,
				IsRead:   notification.IsRead,
			}); err != nil {
				log.Printf("error creating notification: %+v", err)
			}
		case <-ctx.Done():
			break
		}
	}
}

func (me *App) handleGetAllNotifications(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)

	notificationsCount, err := me.queries.GetNotificationsCount(context.Background(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting notifications count: %+v", err))
	}

	page := c.QueryInt("page")
	if page < 1 {
		page = 1
	}

	totalPages := math.Ceil(float64(notificationsCount) / pageSize)
	if page > int(totalPages) {
		return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
			Payload:    []any{},
			TotalPages: int(totalPages),
		})
	}

	offset := (page - 1) * pageSize
	notifications, err := me.queries.GetNotifications(context.Background(), repositry.GetNotificationsParams{
		UserID: userID,
		Limit:  pageSize,
		Offset: int32(offset),
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting notifications: %+v", err))
	}

	payload := make([]types.NotificationPayload, 0, len(notifications))
	for _, notification := range notifications {
		payload = append(payload, types.NotificationPayload{
			ID:        notification.ID,
			Kind:      notification.Kind,
			UserID:    notification.UserID,
			SenderID:  notification.SenderID,
			PostID:    notification.PostID,
			IsRead:    notification.IsRead,
			CreatedAt: notification.CreatedAt,
		})
	}

	return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
		Payload:    payload,
		TotalPages: int(totalPages),
	})
}

func (me *App) handleGetUnreadNotificationsCount(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)

	unreadNotificationsCount, err := me.queries.GetUnreadNotificationsCount(context.Background(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting unread notifications count: %+v", err))
	}

	return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
		Payload: unreadNotificationsCount,
	})
}

func (me *App) handleMarkNotificationAsRead(c *fiber.Ctx) error {
	notificatoinID := c.Params("notification_id")
	userID := getUserIDFromContext(c)

	if exists, err := me.queries.CheckNotificationForUser(context.Background(), repositry.CheckNotificationForUserParams{
		ID:     notificatoinID,
		UserID: userID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking notification: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "notification not found")
	}

	if err := me.queries.MarkNotificationAsRead(context.Background(), notificatoinID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error marking notification as read: %+v", err))
	}

	return c.Status(fiber.StatusOK).SendString("notification marked as read successfully")
}

// getUserIDFromContext retrieves the user ID from the context, which is set by the authentication middleware.
// The user ID is stored in the context under the key "userID" and is expected to be a string.
func getUserIDFromContext(c *fiber.Ctx) string {
	return c.Locals("userID").(string)
}

// parseAndValidateJsonBody parses the JSON request body into `out` and validates it.
// Returns an error if parsing or validation fails.
func parseAndValidateJsonBody(c *fiber.Ctx, out any) error {
	if err := c.BodyParser(out); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid json body")
	}
	if err := utils.ValidateStruct(out); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, fmt.Sprintf("invalid request data: %+v", err))
	}
	return nil
}

func fillUserPayload(userPayload *types.UserPayload, repoUser *repositry.User) {
	userPayload.ID = repoUser.ID
	userPayload.Name = repoUser.Name
	userPayload.Username = repoUser.Username
	userPayload.JoinedAt = repoUser.JoinedAt
	userPayload.PostsCount = repoUser.PostsCount
	userPayload.FollowingCount = repoUser.FollowingCount
	userPayload.FollowersCount = repoUser.FollowersCount
	userPayload.ProfileImageUrl = repoUser.ProfileImageUrl.String
}

func fillPostPayload(postPayload *types.PostPayload, repoPost *repositry.Post) {
	postPayload.ID = repoPost.ID
	postPayload.UserID = repoPost.UserID
	postPayload.Title = repoPost.Title
	postPayload.Content = repoPost.Content
	postPayload.CreatedAt = repoPost.CreatedAt
	postPayload.ViewsCount = repoPost.ViewsCount
	postPayload.CommentsCount = repoPost.CommentsCount
	postPayload.FeaturedImageUrl = repoPost.FeaturedImageUrl.String
}

func fillPostReactions(postPayload *types.PostPayload, repoReactions []repositry.GetPostReactionsRow) {
	for _, reaction := range repoReactions {
		postPayload.Reactions = append(postPayload.Reactions, types.PostPayloadReaction{
			Name:  reaction.Name,
			Count: reaction.Count,
		})
	}
	// still nil if repoReactions is empty
	if postPayload.Reactions == nil {
		postPayload.Reactions = []types.PostPayloadReaction{}
	}
}

func fillCommentPayload(commentPayload *types.CommentPayload, repoComment *repositry.PostComment) {
	commentPayload.ID = repoComment.ID
	commentPayload.PostID = repoComment.PostID
	commentPayload.UserID = repoComment.UserID
	commentPayload.Content = repoComment.Content
	commentPayload.CreatedAt = repoComment.CreatedAt
}
