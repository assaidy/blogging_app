-- name: CreateNotification :one
INSERT INTO notifications(id, kind_id, user_id, sender_id, post_id, is_read)
VALUES($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetNotificationsCount :one
SELECT COUNT(*) FROM notifications
WHERE user_id = $1;

-- name: GetUnreadNotificationsCount :one
SELECT COUNT(*) FROM notifications
WHERE user_id = $1 AND is_read = false;

-- name: GetAllNotifications :many
SELECT 
    n.id,
    nk.name as kind,
    n.user_id,
    n.sender_id,
    n.post_id,
    n.is_read,
    n.created_at
FROM notifications n
JOIN notification_kinds nk ON nk.id = n.kind_id
WHERE
    -- filter
    n.user_id = $1 AND
    -- cursor
    (sqlc.arg(ID)::VARCHAR = '' OR n.id <= sqlc.arg(ID)::VARCHAR)
ORDER BY n.id DESC
LIMIT $2;

-- name: CheckNotificationForUser :one
SELECT EXISTS(SELECT 1 FROM notifications WHERE id = $1 AND user_id = $2);

-- name: MarkNotificationAsRead :exec
UPDATE notifications SET is_read = true WHERE id = $1;
