package types

import (
	"time"
)

type ApiResponse struct {
	Payload      any    `json:"payload,omitempty"`
	TotalPages   int    `json:"totalPages,omitempty"`
	AccessToken  string `json:"accessToken,omitempty"`
	RefreshToken string `json:"refreshToken,omitempty"`
}

type UserPayload struct {
	ID              string    `json:"id"`
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
	ID               string                `json:"id"`
	UserID           string                `json:"userID"`
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
	ID        string    `json:"id"`
	PostID    string    `json:"postID"`
	UserID    string    `json:"userID"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

type NotificationPayload struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	UserID    string    `json:"userID"`
	SenderID  any       `json:"senderID,omitempty"` // nullable string
	PostID    any       `json:"postID,omitempty"`   // nullable string
	IsRead    bool      `json:"isRead"`
	CreatedAt time.Time `json:"createdAt"`
}
