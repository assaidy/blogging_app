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

-- name: GetFollowers :many
SELECT users.*
FROM follows
JOIN users ON follows.follower_id = users.id
WHERE followed_id = $1
ORDER by follows.created_at
LIMIT $2
OFFSET $3;

-- name: GetAllUsers :many
SELECT *
FROM users
WHERE
    (sqlc.arg(FollowersCount)::INTEGER = 0 OR followers_count <= sqlc.arg(FollowersCount)::INTEGER) AND
    (sqlc.arg(PostsCount)::INTEGER = 0 OR posts_count <= sqlc.arg(PostsCount)::INTEGER) AND
    (sqlc.arg(ID)::VARCHAR = '' OR ID <= sqlc.arg(ID)::VARCHAR)
ORDER BY
    followers_count DESC,
    posts_count DESC,
    id DESC
LIMIT $1;
