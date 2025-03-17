package handler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/assaidy/blogging_app/internal/repo/postgres_repo"
	"github.com/assaidy/blogging_app/internal/types"
	"github.com/assaidy/blogging_app/internal/utils"
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
	notificationID := c.Params("notification_id")
	if !utils.IsValidEncodedULID(notificationID) {
		return fiber.NewError(fiber.StatusBadRequest, "invalid ID fromat")
	}
	userID := getUserIDFromContext(c)

	if exists, err := queries.CheckNotificationForUser(context.Background(), postgres_repo.CheckNotificationForUserParams{
		ID:     notificationID,
		UserID: userID,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error checking notification: %+v", err))
	} else if !exists {
		return fiber.NewError(fiber.StatusNotFound, "notification not found")
	}

	if err := queries.MarkNotificationAsRead(context.Background(), notificationID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error marking notification as read: %+v", err))
	}

	return c.Status(fiber.StatusOK).SendString("notification marked as read successfully")
}

func HandleGetAllNotifications(c *fiber.Ctx) error {
	limit := c.QueryInt("limit")
	if limit < 10 || limit > 100 {
		limit = 10
	}

	var requestCursor NotificationsCursor
	if err := decodeBase64AndUnmarshalJson(&requestCursor, c.Query("cursor")); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid cursor format")
	}

	notifications, err := queries.GetAllNotifications(context.Background(), postgres_repo.GetAllNotificationsParams{
		// filter
		UserID: getUserIDFromContext(c),
		// cursor
		ID: requestCursor.ID,
		// limit
		Limit: int32(limit) + 1,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error getting notifications: %+v", err))
	}

	var encodedResponseCursor string
	hasMore := limit < len(notifications)
	if hasMore {
		responseCursor := NotificationsCursor{
			ID: notifications[limit].ID,
		}
		encodedResponseCursor, err = marshalJsonAndEncodeBase64(responseCursor)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("error encoding cursor: %+v", err))
		}
		notifications = notifications[:limit]
	}

	payload := make([]types.NotificationPayload, 0, len(notifications))
	for _, notification := range notifications {
		var notificationPayload types.NotificationPayload
		fillNotificationPayload(&notificationPayload, &notification)
		payload = append(payload, notificationPayload)
	}

	return c.Status(fiber.StatusOK).JSON(CursoredApiResponse{
		Payload:    payload,
		Cursor:     encodedResponseCursor,
		HasMore:    hasMore,
		TotalCount: len(payload),
	})
}
