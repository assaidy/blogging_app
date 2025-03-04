package server

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/assaidy/blogging_app/internal/repositry"
	"github.com/gofiber/fiber/v2"
)

type App struct {
	addr             string
	router           *fiber.App
	queries          *repositry.Queries
	notificationChan chan repositry.Notification
}

func NewAppServer(addr string, queries *repositry.Queries) App {
	server := App{
		router: fiber.New(fiber.Config{
			Prefork:      true,
			ErrorHandler: customErrorHandler,
		}),
		addr:    addr,
		queries: queries,
	}
	return server
}

func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		code = fiberErr.Code
	}
	c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
	// NOTE: Logging occurs before this error handler is executed, so the internal error
	// has already been logged. We avoid exposing internal error details to the client
	// by returning a generic error message.
	if code == fiber.StatusInternalServerError {
		return c.SendStatus(code)
	}
	return c.Status(code).SendString(err.Error())
}

func (me *App) bootstrap(ctx context.Context) {
	me.registerRoutes()
	go me.StartNotificationWorker(ctx)
}

func (me *App) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// bootstrap the app
	me.bootstrap(ctx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	listenErrChan := make(chan error, 1)
	go func() {
		if err := me.router.Listen(me.addr); err != nil {
			listenErrChan <- err
			sigChan <- syscall.SIGINT
		}
	}()

	select {
	case <-sigChan:
	case err := <-listenErrChan:
		return err
	}

	// NOTE: the parent ctx will cancel it anyway
	shutdownCtx, _ := context.WithTimeout(ctx, 5*time.Second)
	if err := me.router.ShutdownWithContext(shutdownCtx); err != nil {
		return err
	}

	log.Printf("server shutdown completed gracefully (PID: %d)", os.Getpid())
	return nil
}
