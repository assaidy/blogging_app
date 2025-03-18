package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/assaidy/blogging_app/internal/db/postgres_db"
	"github.com/assaidy/blogging_app/internal/middleware"
	"github.com/assaidy/blogging_app/internal/repo/postgres_repo"
	"github.com/assaidy/blogging_app/internal/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

var queries = postgres_repo.New(postgres_db.DB)

type ApiResponse struct {
	Payload      any    `json:"payload,omitempty"`
	AccessToken  string `json:"accessToken,omitempty"`
	RefreshToken string `json:"refreshToken,omitempty"`
}

type CursoredApiResponse struct {
	Payload    any    `json:"payload"`
	Cursor     string `json:"cursor"`
	HasMore    bool   `json:"hasNext"`
	TotalCount int    `json:"totalCount"`
}

type UsersCursor struct {
	FollowersCount int       `json:"followersCount"`
	PostsCount     int       `json:"postsCount"`
	ID             uuid.UUID `json:"id" validate:"uuid"`
}

type FollowersCursor struct {
	ID uuid.UUID `json:"id" validate:"uuid"`
}

type UserPostsCursor struct {
	ID uuid.UUID `json:"id" validate:"uuid"`
}

// i might also wanna use reactions count
type PostsCursor struct {
	ViewsCount int       `json:"viewsCount"`
	ID         uuid.UUID `json:"id" validate:"uuid"`
}

type CommentsCursor struct {
	ID uuid.UUID `json:"id" validate:"uuid"`
}

type BookmarksCursor struct {
	CreatedAt time.Time `json:"createdAt"`
}

type NotificationsCursor struct {
	ID uuid.UUID `json:"id" validate:"uuid"`
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

type UserPayload struct {
	ID              uuid.UUID `json:"id"`
	Name            string    `json:"name"`
	Username        string    `json:"username"`
	JoinedAt        time.Time `json:"joinedAt"`
	PostsCount      int32     `json:"postsCount"`
	FollowingCount  int32     `json:"followingCount"`
	FollowersCount  int32     `json:"followersCount"`
	ProfileImageUrl string    `json:"profileImageUrl,omitempty"`
}

type UserRegisterRequest struct {
	Name     string `json:"name" validate:"required,customNoOuterSpaces,max=100"`
	Username string `json:"username" validate:"required,customUsername,max=50"`
	Password string `json:"password" validate:"required,customNoOuterSpaces,min=8,max=50"`
}

type UserLoginRequest struct {
	Username string `json:"username" validate:"required,customUsername,max=50"`
	Password string `json:"password" validate:"required,customNoOuterSpaces,min=8,max=50"`
}

type UserUpdateRequest struct {
	Name            string `json:"name" validate:"required,customNoOuterSpaces"`
	Username        string `json:"username" validate:"required,customUsername"`
	OldPassword     string `json:"oldPassword" validate:"required,customNoOuterSpaces"`
	NewPassword     string `json:"newPassword" validate:"required,customNoOuterSpaces"`
	ProfileImageUrl string `json:"profileImageUrl" validate:"customNoOuterSpaces"`
}

type PostCreateOrUpdateRequest struct {
	Title            string `json:"title" validate:"required,customNoOuterSpaces"`
	Content          string `json:"content" validate:"required,customNoOuterSpaces"`
	FeaturedImageUrl string `json:"featuredImageUrl" validate:"customNoOuterSpaces"`
}

type PostPayload struct {
	ID               uuid.UUID             `json:"id"`
	UserID           uuid.UUID             `json:"userID"`
	Title            string                `json:"title"`
	Content          string                `json:"content"`
	CreatedAt        time.Time             `json:"createdAt"`
	ViewsCount       int32                 `json:"viewsCount"`
	Reactions        []PostPayloadReaction `json:"reactions"`
	CommentsCount    int32                 `json:"commentsCount"`
	FeaturedImageUrl string                `json:"featuredImageUrl,omitempty"`
}

type PostPayloadReaction struct {
	Name  string `json:"name"` // the name of the reaction: like, dislike, ...
	Count int64  `json:"count"`
}

type CommentCreateOrUpdateRequest struct {
	Content string `json:"content" validate:"required,customNoOuterSpaces"`
}

type CommentPayload struct {
	ID        uuid.UUID `json:"id"`
	PostID    uuid.UUID `json:"postID"`
	UserID    uuid.UUID `json:"userID"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

type NotificationPayload struct {
	ID        uuid.UUID `json:"id"`
	Kind      string    `json:"kind"`
	UserID    uuid.UUID `json:"userID"`
	SenderID  uuid.UUID `json:"senderID,omitempty"`
	PostID    uuid.UUID `json:"postID,omitempty"`
	IsRead    bool      `json:"isRead"`
	CreatedAt time.Time `json:"createdAt"`
}

func fillUserPayload(userPayload *UserPayload, repoUser *postgres_repo.User) {
	userPayload.ID = repoUser.ID
	userPayload.Name = repoUser.Name
	userPayload.Username = repoUser.Username
	userPayload.JoinedAt = repoUser.JoinedAt
	userPayload.PostsCount = repoUser.PostsCount
	userPayload.FollowingCount = repoUser.FollowingCount
	userPayload.FollowersCount = repoUser.FollowersCount
	userPayload.ProfileImageUrl = repoUser.ProfileImageUrl.String
}

func fillPostPayload(postPayload *PostPayload, repoPost *postgres_repo.Post) {
	postPayload.ID = repoPost.ID
	postPayload.UserID = repoPost.UserID
	postPayload.Title = repoPost.Title
	postPayload.Content = repoPost.Content
	postPayload.CreatedAt = repoPost.CreatedAt
	postPayload.ViewsCount = repoPost.ViewsCount
	postPayload.CommentsCount = repoPost.CommentsCount
	postPayload.FeaturedImageUrl = repoPost.FeaturedImageUrl.String
}

func fillPostReactions(postPayload *PostPayload, repoReactions []postgres_repo.GetPostReactionsRow) {
	for _, reaction := range repoReactions {
		postPayload.Reactions = append(postPayload.Reactions, PostPayloadReaction{
			Name:  reaction.Name,
			Count: reaction.Count,
		})
	}
	// still nil if repoReactions is empty
	if postPayload.Reactions == nil {
		postPayload.Reactions = []PostPayloadReaction{}
	}
}

func fillCommentPayload(commentPayload *CommentPayload, repoComment *postgres_repo.PostComment) {
	commentPayload.ID = repoComment.ID
	commentPayload.PostID = repoComment.PostID
	commentPayload.UserID = repoComment.UserID
	commentPayload.Content = repoComment.Content
	commentPayload.CreatedAt = repoComment.CreatedAt
}

func fillNotificationPayload(notificationPayload *NotificationPayload, repoNotification *postgres_repo.GetAllNotificationsRow) {
	notificationPayload.ID = repoNotification.ID
	notificationPayload.Kind = repoNotification.Kind
	notificationPayload.UserID = repoNotification.UserID
	notificationPayload.SenderID = repoNotification.SenderID.UUID
	notificationPayload.PostID = repoNotification.PostID.UUID
	notificationPayload.IsRead = repoNotification.IsRead
	notificationPayload.CreatedAt = repoNotification.CreatedAt
}

// getUserIDFromContext retrieves the user ID from the context, which is set by the authentication middleware.
// The user ID is stored in the context under the key "userID" and is expected to be a string.
func getUserIDFromContext(c *fiber.Ctx) uuid.UUID {
	return c.Locals(middleware.AuthUserID).(uuid.UUID)
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
