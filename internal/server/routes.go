package server

import (
	"github.com/assaidy/blogging_app/internal/utils"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func (me *App) registerRoutes() {
	var (
		mwLogger = logger.New()
		mwAuth   = utils.JwtMiddleware(me.queries)
	)

	api := me.router.Group("api", mwLogger)

	v1 := api.Group("v1")
	{
		v1.Post("/auth/register", me.handleRegister)
		v1.Post("/auth/login", me.handleLogin)
		v1.Get("/auth/access_tokens", me.handleGetAccessToken)

		v1.Get("/users/id/:user_id", me.handleGetUserById)
		v1.Get("/users/username/:username", me.handleGetUserByUsername)
		v1.Put("/users", mwAuth, me.handleUpdateUser) // gets id from context
		v1.Delete("/users", mwAuth, me.handleDeleteUser)
		// TODO: get all users with: filteration -> sorting -> pagination

		v1.Post("/follow/:followed_id", mwAuth, me.handleFollow)
		v1.Post("/unfollow/:followed_id", mwAuth, me.handleUnfollow)
		// v1.Get("/users/:user_id/followers", mwAuth, me.handleGetAllFollowers)

		// TEST:
		v1.Post("/posts", mwAuth, me.handleCreatePost)
		v1.Get("/posts/:post_id", me.handleGetPost)
		v1.Put("/posts/:post_id", mwAuth, me.handleUpdatePost)
		v1.Delete("/posts/:post_id", mwAuth, me.handleDeletePost)
		// v1.Get("users/:user_id/posts", me.handleGetAllUserPosts)
		// TODO: get all posts with: filteration -> sorting -> pagination

		v1.Post("/posts/:post_id/views", mwAuth, me.handleViewPost)
		// v1.Get("/posts/:post_id/views", mwAuth, me.handleGetAllPostViews)

		v1.Post("/posts/:post_id/comments", mwAuth, me.handleCreateComment)
		v1.Put("/posts/comments/:comment_id", mwAuth, me.handleUpdateComment)
		v1.Delete("/posts/comments/:comment_id", mwAuth, me.handleDeleteComment)
		// v1.Get("/posts/post_id/comments", mwAuth, me.handleGetAllPostComments)

		v1.Post("/posts/:post_id/like", mwAuth, me.handleReact)
		v1.Delete("/posts/:post_id/reaction", mwAuth, me.handleDeleteReaction)

		v1.Post("/bookmarks/post/:post_id", mwAuth, me.handleAddToBookmarks)
		v1.Delete("/bookmarks/post/:post_id", mwAuth, me.handleDeleteFromBookmarks)
		// v1.Get("/bookmarks", mwAuth, me.handleGetAllBookmarks)

		// v1.Get("/notifications", mwAuth, me.handleGetAllNotifications)
		v1.Get("/notifications/unread_count", mwAuth, me.handleGetUnreadNotificationsCount)
		v1.Post("/notifications/:notification_id/read", mwAuth, me.handleMarkNotificationAsRead)
	}
}
