package handler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/assaidy/blogging_app/internal/repo"
	"github.com/assaidy/blogging_app/internal/repo/postgres_repo"
	"github.com/assaidy/blogging_app/internal/types"
	"github.com/gofiber/fiber/v2"
	"github.com/oklog/ulid/v2"
)

func HandleCreatePost(c *fiber.Ctx) error {
	req := types.PostCreateOrUpdateRequest{}
	if err := parseAndValidateJsonBody(c, &req); err != nil {
		return err
	}

	userID := getUserIDFromContext(c)

	post, err := queries.CreatePost(context.Background(), postgres_repo.CreatePostParams{
		ID:               ulid.Make().String(),
		UserID:           userID,
		Title:            req.Title,
		Content:          req.Content,
		FeaturedImageUrl: sql.NullString{Valid: true, String: req.FeaturedImageUrl},
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error creating post: %+v", err))
	}

	followersIDs, err := queries.GetAllFollowersIDs(context.Background(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting followers IDs: %+v", err))
	}
	for _, id := range followersIDs {
		notificationChan <- postgres_repo.Notification{
			ID:       ulid.Make().String(),
			KindID:   repo.NotificationKindNewPost,
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

func HandleGetPost(c *fiber.Ctx) error {
	postID := c.Params("post_id")

	post, err := queries.GetPost(context.Background(), postID)
	if err != nil {
		if repo.IsNotFoundError(err) {
			return fiber.NewError(fiber.StatusNotFound, "post not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting post: %+v", err))
	}

	reactions, err := queries.GetPostReactions(context.Background(), postID)
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

func HandleUpdatePost(c *fiber.Ctx) error {
	req := types.PostCreateOrUpdateRequest{}
	if err := parseAndValidateJsonBody(c, &req); err != nil {
		return err
	}

	postID := c.Params("post_id")

	if exists, err := queries.CheckPost(context.Background(), postID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking post: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "post not found")
	}

	userID := getUserIDFromContext(c)

	if ok, err := queries.CheckUserOwnsPost(context.Background(), postgres_repo.CheckUserOwnsPostParams{
		ID:     postID,
		UserID: userID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking user owns post: %+v", err))
	} else if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "you don't own this post")
	}

	newPost, err := queries.UpdatePost(context.Background(), postgres_repo.UpdatePostParams{
		ID:               postID,
		Title:            req.Title,
		Content:          req.Content,
		FeaturedImageUrl: sql.NullString{Valid: true, String: req.FeaturedImageUrl},
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error updating post: %+v", err))
	}

	reactions, err := queries.GetPostReactions(context.Background(), postID)
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

func HandleDeletePost(c *fiber.Ctx) error {
	postID := c.Params("post_id")

	if exists, err := queries.CheckPost(context.Background(), postID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking post: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "post not found")
	}

	userID := getUserIDFromContext(c)

	if ok, err := queries.CheckUserOwnsPost(context.Background(), postgres_repo.CheckUserOwnsPostParams{
		ID:     postID,
		UserID: userID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking user owns post: %+v", err))
	} else if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "you don't own this post")
	}

	if err := queries.DeletePost(context.Background(), postID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error deleting post: %+v", err))
	}

	return c.Status(fiber.StatusOK).SendString("post deleted successfully")
}

func HandleViewPost(c *fiber.Ctx) error {
	postID := c.Params("post_id")

	if exists, err := queries.CheckPost(context.Background(), postID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking post: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "post not found")
	}

	userID := getUserIDFromContext(c)

	if ok, err := queries.CheckUserOwnsPost(context.Background(), postgres_repo.CheckUserOwnsPostParams{
		ID:     postID,
		UserID: userID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking user owns post: %+v", err))
	} else if ok {
		return fiber.NewError(fiber.StatusForbidden, "we don't count user viewing his own post")
	}

	if err := queries.ViewPost(context.Background(), postgres_repo.ViewPostParams{
		PostID: postID,
		UserID: userID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error viewing a post: %+v", err))
	}

	return c.Status(fiber.StatusOK).SendString("post view was added successfully")
}
func HandleCreateComment(c *fiber.Ctx) error {
	req := types.CommentCreateOrUpdateRequest{}
	if err := parseAndValidateJsonBody(c, &req); err != nil {
		return err
	}

	postID := c.Params("post_id")

	if exists, err := queries.CheckPost(context.Background(), postID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking post: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "post not found")
	}

	userID := getUserIDFromContext(c)

	comment, err := queries.CreateComment(context.Background(), postgres_repo.CreateCommentParams{
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

func HandleUpdateComment(c *fiber.Ctx) error {
	req := types.CommentCreateOrUpdateRequest{}
	if err := parseAndValidateJsonBody(c, &req); err != nil {
		return err
	}

	commentID := c.Params("comment_id")

	if exists, err := queries.CheckComment(context.Background(), commentID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking comment: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "comment not found")
	}

	userID := getUserIDFromContext(c)

	if ok, err := queries.CheckUserOwnsComment(context.Background(), postgres_repo.CheckUserOwnsCommentParams{
		ID:     commentID,
		UserID: userID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking user owns comment: %+v", err))
	} else if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "you don't own this comment")
	}

	newComment, err := queries.UpdateComment(context.Background(), postgres_repo.UpdateCommentParams{
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

func HandleDeleteComment(c *fiber.Ctx) error {
	commentID := c.Params("comment_id")

	if exists, err := queries.CheckComment(context.Background(), commentID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking comment: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "comment not found")
	}

	userID := getUserIDFromContext(c)

	if ok, err := queries.CheckUserOwnsComment(context.Background(), postgres_repo.CheckUserOwnsCommentParams{
		ID:     commentID,
		UserID: userID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking user owns comment: %+v", err))
	} else if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "you don't own this comment")
	}

	if err := queries.DeleteComment(context.Background(), commentID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error deleting post: %+v", err))
	}

	return c.Status(fiber.StatusOK).SendString("comment deleted successfully")
}

func HandleReact(c *fiber.Ctx) error {
	postID := c.Params("post_id")
	reactionKindName := c.Query("reaction_kind")

	if exists, err := queries.CheckPost(context.Background(), postID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking post: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "post not found")
	}

	kindID, err := queries.GetReactionKindIDByName(context.Background(), reactionKindName)
	if err != nil {
		if repo.IsNotFoundError(err) {
			return fiber.NewError(fiber.StatusBadRequest, "invalid reaction kind")
		}
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting reaction kind id: %+v", err))
	}

	userID := getUserIDFromContext(c)

	if err := queries.CreateReaction(context.Background(), postgres_repo.CreateReactionParams{
		PostID: postID,
		UserID: userID,
		KindID: kindID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error creating reaction: %+v", err))
	}

	return c.Status(fiber.StatusCreated).SendString("reaction added successfully")
}

func HandleDeleteReaction(c *fiber.Ctx) error {
	postID := c.Params("post_id")
	userID := getUserIDFromContext(c)

	if exists, err := queries.CheckReaction(context.Background(), postgres_repo.CheckReactionParams{
		PostID: postID,
		UserID: userID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking reaction: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "you have no reactions on this post")
	}

	if err := queries.DeleteReaction(context.Background(), postgres_repo.DeleteReactionParams{
		PostID: postID,
		UserID: userID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error deleting reaction: %+v", err))
	}

	return c.Status(fiber.StatusOK).SendString("reaction deleted successfully")
}

func HandleAddToBookmarks(c *fiber.Ctx) error {
	postID := c.Params("post_id")

	if exists, err := queries.CheckPost(context.Background(), postID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking post: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "post not found")
	}

	userID := getUserIDFromContext(c)

	if exists, err := queries.CheckBookmark(context.Background(), postgres_repo.CheckBookmarkParams{
		PostID: postID,
		UserID: userID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking bookmark: %+v", err))
	} else if exists {
		return fiber.NewError(fiber.StatusConflict, "bookmarks already exists")
	}

	if err := queries.CreateBookmark(context.Background(), postgres_repo.CreateBookmarkParams{
		PostID: postID,
		UserID: userID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error creating bookmark: %+v", err))
	}

	return c.Status(fiber.StatusCreated).SendString("bookmark created successfully")
}

func HandleDeleteFromBookmarks(c *fiber.Ctx) error {
	postID := c.Params("post_id")
	userID := getUserIDFromContext(c)

	if exists, err := queries.CheckBookmark(context.Background(), postgres_repo.CheckBookmarkParams{
		PostID: postID,
		UserID: userID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking bookmark: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "bookmark doesn't exist")
	}

	if err := queries.DeleteBookmark(context.Background(), postgres_repo.DeleteBookmarkParams{
		PostID: postID,
		UserID: userID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error deleting bookmark: %+v", err))
	}

	return c.Status(fiber.StatusOK).SendString("bookmark deleted successfully")
}
