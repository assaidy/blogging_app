package handler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/assaidy/blogging_app/internal/repo"
	"github.com/assaidy/blogging_app/internal/repo/postgres_repo"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func HandleCreatePost(c *fiber.Ctx) error {
	req := PostCreateOrUpdateRequest{}
	if err := parseAndValidateJsonBody(c, &req); err != nil {
		return err
	}

	userID := getUserIDFromContext(c)

	post, err := queries.CreatePost(context.Background(), postgres_repo.CreatePostParams{
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
			KindID:   repo.NotificationKindNewPost,
			UserID:   id,
			SenderID: uuid.NullUUID{Valid: true, UUID: post.UserID},
			PostID:   uuid.NullUUID{Valid: true, UUID: post.ID},
		}
	}

	var payload PostPayload
	fillPostPayload(&payload, &post)
	payload.Reactions = []PostPayloadReaction{}

	return c.Status(fiber.StatusCreated).JSON(ApiResponse{
		Payload: payload,
	})
}

func HandleGetPost(c *fiber.Ctx) error {
	postID, err := uuid.Parse(c.Params("post_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid ID fromat")
	}

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

	var payload PostPayload
	fillPostPayload(&payload, &post)
	fillPostReactions(&payload, reactions)

	return c.Status(fiber.StatusOK).JSON(ApiResponse{
		Payload: payload,
	})
}

func HandleUpdatePost(c *fiber.Ctx) error {
	req := PostCreateOrUpdateRequest{}
	if err := parseAndValidateJsonBody(c, &req); err != nil {
		return err
	}

	postID, err := uuid.Parse(c.Params("post_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid ID fromat")
	}

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

	var payload PostPayload
	fillPostPayload(&payload, &newPost)
	fillPostReactions(&payload, reactions)

	return c.Status(fiber.StatusOK).JSON(ApiResponse{
		Payload: payload,
	})
}

func HandleDeletePost(c *fiber.Ctx) error {
	postID, err := uuid.Parse(c.Params("post_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid ID fromat")
	}

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
	postID, err := uuid.Parse(c.Params("post_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid ID fromat")
	}

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
	req := CommentCreateOrUpdateRequest{}
	if err := parseAndValidateJsonBody(c, &req); err != nil {
		return err
	}

	postID, err := uuid.Parse(c.Params("post_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid ID fromat")
	}

	if exists, err := queries.CheckPost(context.Background(), postID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking post: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "post not found")
	}

	userID := getUserIDFromContext(c)

	comment, err := queries.CreateComment(context.Background(), postgres_repo.CreateCommentParams{
		PostID:  postID,
		UserID:  userID,
		Content: req.Content,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error creating error: %+v", err))
	}

	var payload CommentPayload
	fillCommentPayload(&payload, &comment)

	return c.Status(fiber.StatusCreated).JSON(ApiResponse{
		Payload: payload,
	})
}

func HandleUpdateComment(c *fiber.Ctx) error {
	req := CommentCreateOrUpdateRequest{}
	if err := parseAndValidateJsonBody(c, &req); err != nil {
		return err
	}

	commentID, err := uuid.Parse(c.Params("comment_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid ID fromat")
	}

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

	var payload CommentPayload
	fillCommentPayload(&payload, &newComment)

	return c.Status(fiber.StatusOK).JSON(ApiResponse{
		Payload: payload,
	})
}

func HandleDeleteComment(c *fiber.Ctx) error {
	commentID, err := uuid.Parse(c.Params("comment_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid ID fromat")
	}

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
	postID, err := uuid.Parse(c.Params("post_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid ID fromat")
	}
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
	postID, err := uuid.Parse(c.Params("post_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid ID fromat")
	}
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
	postID, err := uuid.Parse(c.Params("post_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid ID fromat")
	}

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
	postID, err := uuid.Parse(c.Params("post_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid ID fromat")
	}
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

func HandleGetAllUserPosts(c *fiber.Ctx) error {
	userID, err := uuid.Parse(c.Params("user_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid ID fromat")
	}

	if exists, err := queries.CheckUserID(context.Background(), userID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking user: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}

	limit := c.QueryInt("limit")
	if limit < 10 || limit > 100 {
		limit = 10
	}

	var requestCursor UserPostsCursor
	if err := decodeBase64AndUnmarshalJson(&requestCursor, c.Query("cursor")); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid cursor format")
	}

	posts, err := queries.GetAllUserPosts(context.Background(), postgres_repo.GetAllUserPostsParams{
		// filter
		UserID: userID,
		// cursor
		ID: requestCursor.ID,
		// limit
		Limit: int32(limit) + 1,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting user posts: %+v", err))
	}

	var encodedResponseCursor string
	hasMore := limit < len(posts)
	if hasMore {
		responseCursor := UserPostsCursor{
			ID: posts[limit].ID,
		}
		encodedResponseCursor, err = marshalJsonAndEncodeBase64(responseCursor)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error encoding cursor: %+v", err))
		}
		posts = posts[:limit]
	}

	payload := make([]PostPayload, 0, len(posts))
	for _, post := range posts {
		var postPayload PostPayload
		fillPostPayload(&postPayload, &post)
		payload = append(payload, postPayload)
	}

	return c.Status(fiber.StatusOK).JSON(CursoredApiResponse{
		Payload:    payload,
		Cursor:     encodedResponseCursor,
		HasMore:    hasMore,
		TotalCount: len(payload),
	})
}

func HandleGetAllPosts(c *fiber.Ctx) error {
	limit := c.QueryInt("limit")
	if limit < 10 || limit > 100 {
		limit = 10
	}

	var requestCursor PostsCursor
	if err := decodeBase64AndUnmarshalJson(&requestCursor, c.Query("cursor")); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid cursor format")
	}

	posts, err := queries.GetAllPosts(context.Background(), postgres_repo.GetAllPostsParams{
		// filter
		SearchQuery: c.Query("search_query"),
		// cursor
		ViewsCount: int32(requestCursor.ViewsCount),
		ID:         requestCursor.ID,
		// limit
		Limit: int32(limit) + 1,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting posts: %+v", err))
	}

	var encodedResponseCursor string
	hasMore := limit < len(posts)
	if hasMore {
		responseCursor := PostsCursor{
			ViewsCount: int(posts[limit].ViewsCount),
			ID:         posts[limit].ID,
		}
		encodedResponseCursor, err = marshalJsonAndEncodeBase64(responseCursor)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error encoding cursor: %+v", err))
		}
		posts = posts[:limit]
	}

	payload := make([]PostPayload, 0, len(posts))
	for _, post := range posts {
		var postPayload PostPayload
		fillPostPayload(&postPayload, &post)
		payload = append(payload, postPayload)
	}

	return c.Status(fiber.StatusOK).JSON(CursoredApiResponse{
		Payload:    payload,
		Cursor:     encodedResponseCursor,
		HasMore:    hasMore,
		TotalCount: len(payload),
	})
}

func HandleGetAllPostComments(c *fiber.Ctx) error {
	postID, err := uuid.Parse(c.Params("post_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid ID fromat")
	}

	if exists, err := queries.CheckPost(context.Background(), postID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking post: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "post not found")
	}

	limit := c.QueryInt("limit")
	if limit < 10 || limit > 100 {
		limit = 10
	}

	var requestCursor CommentsCursor
	if err := decodeBase64AndUnmarshalJson(&requestCursor, c.Query("cursor")); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid cursor format")
	}

	comments, err := queries.GetAllPostComments(context.Background(), postgres_repo.GetAllPostCommentsParams{
		// filter
		PostID: postID,
		// cursor
		ID: requestCursor.ID,
		// limit
		Limit: int32(limit) + 1,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting post comments: %+v", err))
	}

	var encodedResponseCursor string
	hasMore := limit < len(comments)
	if hasMore {
		responseCursor := CommentsCursor{
			ID: comments[limit].ID,
		}
		encodedResponseCursor, err = marshalJsonAndEncodeBase64(responseCursor)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error encoding cursor: %+v", err))
		}
		comments = comments[:limit]
	}

	payload := make([]CommentPayload, 0, len(comments))
	for _, comment := range comments {
		var commentPayload CommentPayload
		fillCommentPayload(&commentPayload, &comment)
		payload = append(payload, commentPayload)
	}

	return c.Status(fiber.StatusOK).JSON(CursoredApiResponse{
		Payload:    payload,
		Cursor:     encodedResponseCursor,
		HasMore:    hasMore,
		TotalCount: len(payload),
	})
}

func HandleGetAllBookmarks(c *fiber.Ctx) error {
	limit := c.QueryInt("limit")
	if limit < 10 || limit > 100 {
		limit = 10
	}

	var requestCursor BookmarksCursor
	if err := decodeBase64AndUnmarshalJson(&requestCursor, c.Query("cursor")); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid cursor format")
	}

	bookmarks, err := queries.GetAllBookmarks(context.Background(), postgres_repo.GetAllBookmarksParams{
		// filter
		UserID: getUserIDFromContext(c),
		// cursor
		CreatedAt: requestCursor.CreatedAt,
		// limit
		Limit: int32(limit) + 1,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting bookmarks: %+v", err))
	}

	var encodedResponseCursor string
	hasMore := limit < len(bookmarks)
	if hasMore {
		responseCursor := BookmarksCursor{
			CreatedAt: bookmarks[limit].CreatedAt,
		}
		encodedResponseCursor, err = marshalJsonAndEncodeBase64(responseCursor)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error encoding cursor: %+v", err))
		}
		bookmarks = bookmarks[:limit]
	}

	payload := make([]PostPayload, 0, len(bookmarks))
	for _, bookmark := range bookmarks {
		var postPayload PostPayload
		fillPostPayload(&postPayload, &bookmark)
		payload = append(payload, postPayload)
	}

	return c.Status(fiber.StatusOK).JSON(CursoredApiResponse{
		Payload:    payload,
		Cursor:     encodedResponseCursor,
		HasMore:    hasMore,
		TotalCount: len(payload),
	})
}
