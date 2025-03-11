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
GROUP BY pr.kind_id, rk.name;

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

-- name: ViewPost :exec
INSERT INTO post_views(post_id, user_id)
VALUES($1, $2)
ON CONFLICT(post_id, user_id) DO NOTHING;

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
