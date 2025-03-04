-- ====================================================================================================
-- users
-- ====================================================================================================

-- name: CreateUser :one
INSERT INTO users(id, name, username, hashed_password, profile_image_url)
VALUES($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: CheckUserID :one
SELECT EXISTS(SELECT 1 FROM users WHERE id = $1);

-- name: CheckUsername :one
SELECT EXISTS(SELECT 1 FROM users WHERE username = $1);

-- name: UpdateUser :one
UPDATE users
SET 
    name = $1,
    username = $2,
    hashed_password = $3,
    profile_image_url = $4
WHERE id = $5
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- ====================================================================================================
-- follows
-- ====================================================================================================

-- name: CreateFollow :exec
INSERT INTO follows(follower_id, followed_id)
VALUES($1, $2)
ON CONFLICT(follower_id, followed_id) DO NOTHING;

-- name: DeleteFollow :exec
DELETE FROM follows WHERE follower_id = $1 AND followed_id = $2;

-- name: GetAllFollowersIDs :many
SELECT follower_id FROM follows WHERE followed_id = $1;

-- name: GetFollowersCount :one
SELECT COUNT(*) FROM follows WHERE followed_id = $1;

-- name: GetFollowers :many
SELECT users.*
FROM follows
JOIN users ON follows.follower_id = users.id
WHERE followed_id = $1
ORDER by follows.created_at
LIMIT $2
OFFSET $3;

-- ====================================================================================================
-- tokens
-- ====================================================================================================

-- name: CreateRefreshToken :exec
INSERT INTO refresh_tokens(token, user_id, expires_at)
VALUES($1, $2, $3);

-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens WHERE token = $1 AND user_id = $2;

-- name: DeleteRefreshToken :exec
DELETE FROM refresh_tokens WHERE token = $1;

-- ====================================================================================================
-- posts
-- ====================================================================================================
-- name: CreatePost :one
INSERT INTO posts(id, user_id, title, content, featured_image_url)
VALUES($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetPost :one
SELECT * FROM posts WHERE id = $1;

-- name: GetPostReactions :many
SELECT 
    rk.name,
    COUNT(pr.kind_id) AS count
FROM post_reactions pr
JOIN reaction_kinds rk ON pr.kind_id = rk.id
WHERE pr.post_id = $1
GROUP BY pr.kind_id;

-- name: GetReactionKindIDByName :one
SELECT id FROM reaction_kinds WHERE name = $1;

-- name: CreateReaction :exec
INSERT INTO post_reactions(post_id, user_id, kind_id)
VALUES($1, $2, $3)
ON CONFLICT(post_id, user_id) DO UPDATE
SET kind_id = EXCLUDED.kind_id;

-- name: CheckPost :one
SELECT EXISTS(select 1 FROM posts WHERE id = $1);

-- name: CheckUserOwnsPost :one
SELECT EXISTS(select 1 FROM posts WHERE id = $1 AND user_id = $2);

-- name: UpdatePost :one
UPDATE posts
SET 
    title = $1,
    content = $2,
    featured_image_url = $3
WHERE id = $4
RETURNING *;

-- name: DeletePost :exec
DELETE FROM posts WHERE id = $1;

-- name: GetUserPostsCount :one
SELECT posts_count FROM users WHERE id = $1;

-- name: GetUserPosts :many
SELECT *
FROM posts
WHERE user_id = $1
ORDER BY created_at
LIMIT $2
OFFSET $3;

-- name: GetPostViewsCount :one
SELECT views_count FROM posts WHERE id = $1;

-- name: GetPostViews :many
SELECT users.*
FROM post_views
JOIN users ON post_views.user_id = users.id
WHERE post_id = $1
ORDER BY post_views.created_at
LIMIT $2
OFFSET $3;

-- ====================================================================================================
-- views
-- ====================================================================================================

-- name: ViewPost :exec
INSERT INTO post_views(post_id, user_id)
VALUES($1, $2)
ON CONFLICT(post_id, user_id) DO NOTHING;

-- ====================================================================================================
-- comments
-- ====================================================================================================

-- name: CreateComment :one
INSERT INTO post_comments(id, post_id, user_id, content)
VALUES($1, $2, $3, $4)
RETURNING *;

-- name: CheckComment :one
SELECT EXISTS(select 1 FROM post_comments WHERE id = $1);

-- name: CheckUserOwnsComment :one
SELECT EXISTS(select 1 FROM post_comments WHERE id = $1 AND user_id = $2);

-- name: UpdateComment :one
UPDATE post_comments
SET content = $1
WHERE id = $2
RETURNING *;

-- name: DeleteComment :exec
DELETE FROM post_comments WHERE id = $1;

-- name: GetPostCommentsCount :one
SELECT comments_count FROM posts WHERE id = $1;

-- name: GetPostComments :many
SELECT *
FROM post_comments
WHERE post_id = $1
ORDER BY created_at
LIMIT $2
OFFSET $3;

-- ====================================================================================================
-- reactions
-- ====================================================================================================
-- name: CreateLike :exec
insert into post_reactions(post_id, user_id, kind)
VALUES($1, $2, 'like')
ON CONFLICT(post_id, user_id) DO UPDATE 
SET kind = EXCLUDED.kind, created_at = NOW();

-- name: CreateDislike :exec
insert into post_reactions(post_id, user_id, kind)
VALUES($1, $2, 'dislike')
ON CONFLICT(post_id, user_id) DO UPDATE 
SET kind = EXCLUDED.kind, created_at = NOW();

-- name: CheckReaction :one
SELECT EXISTS(SELECT 1 FROM post_reactions WHERE post_id = $1 AND user_id = $2);

-- name: DeleteReaction :exec
DELETE FROM post_reactions WHERE post_id = $1 AND user_id = $2;

-- ====================================================================================================
-- bookmarks
-- ====================================================================================================

-- name: CreateBookmark :exec
INSERT INTO bookmarks(user_id, post_id)
VALUES($1, $2)
ON CONFLICT(user_id, post_id) DO NOTHING;

-- name: CheckBookmark :one
SELECT EXISTS(SELECT 1 FROM bookmarks WHERE user_id = $1 AND post_id = $2);

-- name: DeleteBookmark :exec
DELETE FROM bookmarks WHERE user_id = $1 AND post_id = $2;

-- name: GetBookmarksCount :one
SELECT COUNT(*) FROM bookmarks WHERE user_id = $1;

-- name: GetBookmarks :many
SELECT posts.*
FROM bookmarks
JOIN posts ON bookmarks.post_id = posts.id
WHERE bookmarks.user_id = $1
ORDER BY bookmarks.created_at
LIMIT $2
OFFSET $3;

-- ====================================================================================================
-- notifications
-- ====================================================================================================

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

-- name: GetNotifications :many
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
WHERE n.user_id = $1
ORDER BY n.created_at DESC
LIMIT $2
OFFSET $3;

-- name: CheckNotificationForUser :one
SELECT EXISTS(SELECT 1 FROM notifications WHERE id = $1 AND user_id = $2);

-- name: MarkNotificationAsRead :exec
UPDATE notifications SET is_read = true WHERE id = $1;
