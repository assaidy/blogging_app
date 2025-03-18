-- name: CreateUser :one
INSERT INTO users(name, username, hashed_password, profile_image_url)
VALUES($1, $2, $3, $4)
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

-- name: CreateFollow :exec
INSERT INTO follows(follower_id, followed_id)
VALUES($1, $2)
ON CONFLICT(follower_id, followed_id) DO NOTHING;

-- name: DeleteFollow :exec
DELETE FROM follows WHERE follower_id = $1 AND followed_id = $2;

-- name: GetAllFollowersIDs :many
SELECT follower_id FROM follows WHERE followed_id = $1;

-- name: CheckFollow :one
SELECT EXISTS(SELECT 1 FROM follows WHERE follower_id = $1 AND followed_id = $2);

-- name: GetFollowersCount :one
SELECT COUNT(*) FROM follows WHERE followed_id = $1;

-- name: GetAllUsers :many
SELECT *
FROM users
WHERE
     -- filter
    (name ILIKE '%' || sqlc.arg(Name)::VARCHAR || '%' OR username ILIKE '%' || sqlc.arg(Username)::VARCHAR || '%') AND
     -- cursor
    (sqlc.arg(followers_count)::INTEGER = 0 OR followers_count <= sqlc.arg(followers_count)::INTEGER) AND
    (sqlc.arg(posts_count)::INTEGER = 0 OR posts_count <= sqlc.arg(posts_count)::INTEGER) AND
    (is_zero_uuid(sqlc.arg(ID)::UUID) OR id <= sqlc.arg(ID)::UUID)
ORDER BY
    followers_count DESC,
    posts_count DESC,
    id DESC
LIMIT $1;

-- name: GetAllFollowers :many
SELECT users.*
FROM follows
JOIN users ON follows.follower_id = users.id
WHERE
    -- filter
    followed_id = $1 AND
    -- cursor
    (is_zero_uuid(sqlc.arg(ID)::UUID) OR id <= sqlc.arg(ID)::UUID)
ORDER BY users.id DESC
LIMIT $2;
