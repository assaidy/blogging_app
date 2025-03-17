package main

import (
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/assaidy/blogging_app/internal/handler"
	"github.com/assaidy/blogging_app/internal/middleware"
	"github.com/gofiber/fiber/v2"
	_ "github.com/joho/godotenv/autoload"
	_ "github.com/lib/pq"
)

func mountRoutes(app *fiber.App) {
	api := app.Group("api", middleware.Logger)

	v1 := api.Group("v1")
	{
		v1.Post("/auth/register", handler.HandleRegister)
		v1.Post("/auth/login", handler.HandleLogin)
		v1.Get("/auth/access_tokens", handler.HandleGetAccessToken)

		v1.Get("/users/id/:user_id", handler.HandleGetUserById)
		v1.Get("/users/username/:username", handler.HandleGetUserByUsername)
		v1.Put("/users", middleware.Auth, handler.HandleUpdateUser)
		v1.Delete("/users", middleware.Auth, handler.HandleDeleteUser)
		v1.Get("/users", middleware.Auth, handler.HandleGetAllUsers) // with filtering (used for searching)

		v1.Post("/follow/:followed_id", middleware.Auth, handler.HandleFollow)
		v1.Post("/unfollow/:followed_id", middleware.Auth, handler.HandleUnfollow)
		v1.Get("/users/:user_id/followers", middleware.Auth, handler.HandleGetAllFollowers)

		v1.Post("/posts", middleware.Auth, handler.HandleCreatePost)
		v1.Get("/posts/:post_id", handler.HandleGetPost)
		v1.Put("/posts/:post_id", middleware.Auth, handler.HandleUpdatePost)
		v1.Delete("/posts/:post_id", middleware.Auth, handler.HandleDeletePost)
		v1.Get("users/:user_id/posts", middleware.Auth, handler.HandleGetAllUserPosts)
		v1.Get("posts", middleware.Auth, handler.HandleGetAllPosts) // with filtering (used for searching)

		v1.Post("/posts/:post_id/views", middleware.Auth, handler.HandleViewPost)

		v1.Post("/posts/:post_id/comments", middleware.Auth, handler.HandleCreateComment)
		v1.Put("/posts/comments/:comment_id", middleware.Auth, handler.HandleUpdateComment)
		v1.Delete("/posts/comments/:comment_id", middleware.Auth, handler.HandleDeleteComment)
		v1.Get("/posts/post_id/comments", middleware.Auth, handler.HandleGetAllPostComments)

		v1.Post("/posts/:post_id/reaction", middleware.Auth, handler.HandleReact)
		v1.Delete("/posts/:post_id/reaction", middleware.Auth, handler.HandleDeleteReaction)

		v1.Post("/bookmarks/post/:post_id", middleware.Auth, handler.HandleAddToBookmarks)
		v1.Delete("/bookmarks/post/:post_id", middleware.Auth, handler.HandleDeleteFromBookmarks)
		v1.Get("/bookmarks", middleware.Auth, handler.HandleGetAllBookmarks)

		v1.Get("/notifications", middleware.Auth, handler.HandleGetAllNotifications)
		v1.Get("/notifications/unread_count", middleware.Auth, handler.HandleGetUnreadNotificationsCount)
		v1.Post("/notifications/:notification_id/read", middleware.Auth, handler.HandleMarkNotificationAsRead)
	}
}

func main() {
	app := fiber.New(fiber.Config{
		AppName:      "blogging app",
		ServerHeader: "blogging app",
		Prefork:      true,
		ErrorHandler: middleware.ErrorHandler,
	})

	// mount routes
	mountRoutes(app)

	// start server listening
	go func() {
		if err := app.Listen(":" + os.Getenv("PORT")); err != nil {
			log.Fatal(err)
		}
	}()

	// start notification worker (notification channel/bus)
	handler.StartNotificationWorkers()
	defer handler.StopNotificationWorker()

	// listen for termination signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	// wait for termination signal
	<-sigChan

	// shutdown server
	if err := app.ShutdownWithTimeout(5 * time.Second); err != nil {
		slog.Error("error shutting down server", "err", err, "pid", os.Getpid())
	} else {
		slog.Info("server shutdown completed gracefully", "pid", os.Getpid())
	}
}
