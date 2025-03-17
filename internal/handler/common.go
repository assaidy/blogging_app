package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/assaidy/blogging_app/internal/db/postgres_db"
	"github.com/assaidy/blogging_app/internal/middleware"
	"github.com/assaidy/blogging_app/internal/repo/postgres_repo"
	"github.com/assaidy/blogging_app/internal/types"
	"github.com/assaidy/blogging_app/internal/utils"
	"github.com/gofiber/fiber/v2"
)

var queries = postgres_repo.New(postgres_db.DB)

type CursoredApiResponse struct {
	Payload    any    `json:"payload"`
	Cursor     string `json:"cursor"`
	HasMore    bool   `json:"hasNext"`
	TotalCount int    `json:"totalCount"`
}

type UsersCursor struct {
	FollowersCount int    `json:"followersCount"`
	PostsCount     int    `json:"postsCount"`
	ID             string `json:"id" validate:"customULID"`
}

type FollowersCursor struct {
	ID string `json:"id" validate:"customULID"`
}

// TODO: might also wanna use reactions count
type PostsCursor struct {
	ViewsCount int    `json:"viewsCount"`
	ID         string `json:"id" validate:"customULID"`
}

type CommentsCursor struct {
	ID string `json:"id" validate:"customULID"`
}

type BookmarksCursor struct {
	CreatedAt time.Time `json:"createdAt"`
}

type NotificationsCursor struct {
	ID string `json:"id" validate:"customULID"`
}

// decodeBase64AndUnmarshalJson decodes a base64-encoded string and unmarshals it into the provided output struct.
// It first checks if the base64String is empty, returning nil if it is. Otherwise, it decodes the base64 string
// into JSON bytes, unmarshals those bytes into the output struct, and then validates the struct using utils.ValidateStruct.
// Returns an error if any step in the process fails.
func decodeBase64AndUnmarshalJson(out any, base64String string) error {
	if base64String == "" {
		return nil
	}
	jsonBytes, err := base64.StdEncoding.DecodeString(base64String)
	if err != nil {
		return err
	}
	err = json.Unmarshal(jsonBytes, out)
	if err != nil {
		return err
	}
	err = utils.ValidateStruct(out)
	return err
}

// marshalJsonAndEncodeBase64 marshals the provided source struct into JSON bytes and then encodes those bytes
// into a base64-encoded string. Returns the base64-encoded string or an error if the marshaling fails.
func marshalJsonAndEncodeBase64(src any) (string, error) {
	jsonBytes, err := json.Marshal(src)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(jsonBytes), nil
}

// getUserIDFromContext retrieves the user ID from the context, which is set by the authentication middleware.
// The user ID is stored in the context under the key "userID" and is expected to be a string.
func getUserIDFromContext(c *fiber.Ctx) string {
	return c.Locals(middleware.AuthUserID).(string)
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

func fillUserPayload(userPayload *types.UserPayload, repoUser *postgres_repo.User) {
	userPayload.ID = repoUser.ID
	userPayload.Name = repoUser.Name
	userPayload.Username = repoUser.Username
	userPayload.JoinedAt = repoUser.JoinedAt
	userPayload.PostsCount = repoUser.PostsCount
	userPayload.FollowingCount = repoUser.FollowingCount
	userPayload.FollowersCount = repoUser.FollowersCount
	userPayload.ProfileImageUrl = repoUser.ProfileImageUrl.String
}

func fillPostPayload(postPayload *types.PostPayload, repoPost *postgres_repo.Post) {
	postPayload.ID = repoPost.ID
	postPayload.UserID = repoPost.UserID
	postPayload.Title = repoPost.Title
	postPayload.Content = repoPost.Content
	postPayload.CreatedAt = repoPost.CreatedAt
	postPayload.ViewsCount = repoPost.ViewsCount
	postPayload.CommentsCount = repoPost.CommentsCount
	postPayload.FeaturedImageUrl = repoPost.FeaturedImageUrl.String
}

func fillPostReactions(postPayload *types.PostPayload, repoReactions []postgres_repo.GetPostReactionsRow) {
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

func fillCommentPayload(commentPayload *types.CommentPayload, repoComment *postgres_repo.PostComment) {
	commentPayload.ID = repoComment.ID
	commentPayload.PostID = repoComment.PostID
	commentPayload.UserID = repoComment.UserID
	commentPayload.Content = repoComment.Content
	commentPayload.CreatedAt = repoComment.CreatedAt
}

func fillNotificationPayload(notificationPayload *types.NotificationPayload, repoNotification *postgres_repo.GetAllNotificationsRow) {
	notificationPayload.ID = repoNotification.ID
	notificationPayload.Kind = repoNotification.Kind
	notificationPayload.UserID = repoNotification.UserID
	notificationPayload.SenderID = repoNotification.SenderID
	notificationPayload.PostID = repoNotification.PostID
	notificationPayload.IsRead = repoNotification.IsRead
	notificationPayload.CreatedAt = repoNotification.CreatedAt
}
