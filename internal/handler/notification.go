package handler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/assaidy/blogging_app/internal/repo/postgres_repo"
	"github.com/assaidy/blogging_app/internal/types"
	"github.com/gofiber/fiber/v2"
)

var (
	notificationChan = make(chan postgres_repo.Notification, 1000)
	stopChan         = make(chan struct{}, 1)
	wg               sync.WaitGroup
)

const numWorkers = 10

func StartNotificationWorkers() {
	for range numWorkers {
		wg.Add(1)

		go func() {
			defer wg.Done()
			for notification := range notificationChan {
				if _, err := queries.CreateNotification(context.Background(), postgres_repo.CreateNotificationParams{
					ID:       notification.ID,
					KindID:   notification.KindID,
					UserID:   notification.UserID,
					SenderID: notification.SenderID,
					PostID:   notification.PostID,
					IsRead:   notification.IsRead,
				}); err != nil {
					slog.Error("error creating notification", "err", err)
				}
			}
		}()
	}
}

func StopNotificationWorker() {
	close(notificationChan)
	wg.Wait()
}

func HandleGetUnreadNotificationsCount(c *fiber.Ctx) error {
	userID := getUserIDFromContext(c)

	unreadNotificationsCount, err := queries.GetUnreadNotificationsCount(context.Background(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting unread notifications count: %+v", err))
	}

	return c.Status(fiber.StatusOK).JSON(types.ApiResponse{
		Payload: unreadNotificationsCount,
	})
}

func HandleMarkNotificationAsRead(c *fiber.Ctx) error {
	notificatoinID := c.Params("notification_id")
	userID := getUserIDFromContext(c)

	if exists, err := queries.CheckNotificationForUser(context.Background(), postgres_repo.CheckNotificationForUserParams{
		ID:     notificatoinID,
		UserID: userID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking notification: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "notification not found")
	}

	if err := queries.MarkNotificationAsRead(context.Background(), notificatoinID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error marking notification as read: %+v", err))
	}

	return c.Status(fiber.StatusOK).SendString("notification marked as read successfully")
}
