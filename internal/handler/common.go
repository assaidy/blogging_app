package handler

import (
	"fmt"

	"github.com/assaidy/blogging_app/internal/db/postgres_db"
	"github.com/assaidy/blogging_app/internal/middleware"
	"github.com/assaidy/blogging_app/internal/repo/postgres_repo"
	"github.com/assaidy/blogging_app/internal/types"
	"github.com/assaidy/blogging_app/internal/utils"
	"github.com/gofiber/fiber/v2"
)

var queries = postgres_repo.New(postgres_db.DB)

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
