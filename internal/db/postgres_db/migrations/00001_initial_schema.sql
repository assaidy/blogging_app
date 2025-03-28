-- +goose Up

-- for gen_random_bytes()
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- +goose StatementBegin
CREATE FUNCTION generate_ulid_as_uuid() -- https://blog.daveallie.com/ulid-primary-keys/
RETURNS UUID
AS $$
    SELECT (
        lpad(to_hex(floor(extract(epoch FROM clock_timestamp()) * 1000)::BIGINT), 12, '0') || encode(gen_random_bytes(10), 'hex')
    )::UUID;
$$ LANGUAGE SQL;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION is_zero_uuid(arg UUID)
RETURNS BOOLEAN
AS $$
    SELECT arg = '00000000-0000-0000-0000-000000000000';
$$ LANGUAGE SQL;
-- +goose StatementEnd

CREATE TABLE users(
    id UUID DEFAULT generate_ulid_as_uuid(),
    name VARCHAR(255) NOT NULL,
    username VARCHAR(50) NOT NULL,
    hashed_password VARCHAR(100) NOT NULL,
    joined_at TIMESTAMP NOT NULL DEFAULT NOW(),
    posts_count INTEGER NOT NULL DEFAULT 0,
    following_count INTEGER NOT NULL DEFAULT 0,
    followers_count INTEGER NOT NULL DEFAULT 0,
    profile_image_url VARCHAR,

    PRIMARY KEY(id),
    UNIQUE(username)
);

CREATE TABLE refresh_tokens(
    token VARCHAR(255),
    user_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,

    PRIMARY KEY(token),
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE follows(
    follower_id UUID,
    followed_id UUID,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    PRIMARY KEY(follower_id, followed_id),
    FOREIGN KEY(follower_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY(followed_id) REFERENCES users(id) ON DELETE CASCADE
);

-- +goose StatementBegin
CREATE FUNCTION update_user_follow_counts()
RETURNS TRIGGER
AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE users SET following_count = following_count + 1 WHERE id = NEW.follower_id;
        UPDATE users SET followers_count = followers_count + 1 WHERE id = NEW.followed_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE users SET following_count = following_count - 1 WHERE id = OLD.follower_id;
        UPDATE users SET followers_count = followers_count - 1 WHERE id = OLD.followed_id;
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE PLPGSQL;
-- +goose StatementEnd

CREATE TRIGGER trg_update_user_follow_counts
AFTER INSERT OR DELETE ON follows FOR EACH ROW
EXECUTE FUNCTION update_user_follow_counts();

CREATE TABLE posts(
    id UUID DEFAULT generate_ulid_as_uuid(),
    user_id UUID NOT NULL,
    title VARCHAR(255) NOT NULL,
    content VARCHAR NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    views_count INTEGER NOT NULL DEFAULT 0,
    comments_count INTEGER NOT NULL DEFAULT 0,
    featured_image_url VARCHAR,

    PRIMARY KEY(id),
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX ON posts(user_id);
-- used for full text search
CREATE INDEX ON posts USING GIN(to_tsvector('english', title || ' ' || content));

-- +goose StatementBegin
CREATE FUNCTION update_user_posts_count()
RETURNS TRIGGER
AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE users SET posts_count = posts_count + 1 WHERE id = NEW.user_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE users SET posts_count = posts_count - 1 WHERE id = OLD.user_id;
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE PLPGSQL;
-- +goose StatementEnd

CREATE TRIGGER trg_update_user_posts_count
AFTER INSERT OR DELETE ON posts FOR EACH ROW
EXECUTE FUNCTION update_user_posts_count();

CREATE TABLE post_views(
    post_id UUID,
    user_id UUID,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    PRIMARY KEY(user_id, post_id),
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY(post_id) REFERENCES posts(id) ON DELETE CASCADE
);

-- +goose StatementBegin
CREATE FUNCTION update_post_views_count()
RETURNS TRIGGER
AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE posts SET views_count = views_count + 1 WHERE id = NEW.post_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE posts SET views_count = views_count - 1 WHERE id = OLD.post_id;
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE PLPGSQL;
-- +goose StatementEnd

CREATE TRIGGER trg_update_post_views_count
AFTER INSERT OR DELETE ON post_views FOR EACH ROW
EXECUTE FUNCTION update_post_views_count();

CREATE TABLE post_comments(
    id UUID DEFAULT generate_ulid_as_uuid(),
    post_id UUID NOT NULL,
    user_id UUID NOT NULL,
    content VARCHAR NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    PRIMARY KEY(id),
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY(post_id) REFERENCES posts(id) ON DELETE CASCADE
);

CREATE INDEX ON post_comments(post_id);

-- +goose StatementBegin
CREATE FUNCTION update_post_comments_count()
RETURNS TRIGGER
AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE posts SET comments_count = comments_count + 1 WHERE id = NEW.post_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE posts SET comments_count = comments_count - 1 WHERE id = OLD.post_id;
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE PLPGSQL;
-- +goose StatementEnd

CREATE TRIGGER trg_update_post_comments_count
AFTER INSERT OR DELETE ON post_comments FOR EACH ROW
EXECUTE FUNCTION update_post_comments_count();

CREATE TABLE reaction_kinds(
    id SERIAL,
    name VARCHAR(50) NOT NULL,

    PRIMARY KEY(id),
    UNIQUE(name)
);

INSERT INTO reaction_kinds(name)
VALUES
    ('like'),
    ('dislike');

CREATE TABLE post_reactions (
    post_id UUID,
    user_id UUID,
    kind_id INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    PRIMARY KEY(post_id, user_id),
    FOREIGN KEY(kind_id) REFERENCES reaction_kinds(id) ON DELETE CASCADE,
    FOREIGN KEY(post_id) REFERENCES posts(id) ON DELETE CASCADE,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE bookmarks(
    user_id UUID,
    post_id UUID,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    PRIMARY KEY(user_id, post_id),
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY(post_id) REFERENCES posts(id) ON DELETE CASCADE
);

CREATE TABLE notification_kinds(
    id SERIAL,
    name VARCHAR(50) NOT NULL, -- 'new_follower', 'new_post', ...

    PRIMARY KEY(id)
);

INSERT INTO notification_kinds(name)
VALUES 
    ('new_follower'),
    ('new_post');

CREATE TABLE notifications(
    id UUID DEFAULT generate_ulid_as_uuid(),
    kind_id INTEGER NOT NULL, 
    user_id UUID NOT NULL, -- the user who recieves the notification
    sender_id UUID, -- the user who trigger the notification
    post_id UUID, -- in case kind is 'new_post'
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    PRIMARY KEY(id),
    FOREIGN KEY(kind_id) REFERENCES notification_kinds(id) ON DELETE CASCADE,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY(sender_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY(post_id) REFERENCES posts(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS refresh_tokens CASCADE;
DROP TABLE IF EXISTS follows CASCADE; 
DROP TABLE IF EXISTS posts CASCADE;
DROP TABLE IF EXISTS post_views CASCADE; 
DROP TABLE IF EXISTS post_comments CASCADE; 
DROP TABLE IF EXISTS reaction_kinds CASCADE; 
DROP TABLE IF EXISTS post_reactions CASCADE; 
DROP TABLE IF EXISTS bookmarks CASCADE; 
DROP TABLE IF EXISTS notification_kinds CASCADE; 
DROP TABLE IF EXISTS notifications CASCADE; 
-- DROP DOMAIN IF EXISTS ulid;

DROP FUNCTION IF EXISTS update_user_posts_count;
DROP FUNCTION IF EXISTS update_user_follow_counts;
DROP FUNCTION IF EXISTS update_post_views_count;
DROP FUNCTION IF EXISTS update_post_comments_count;
DROP FUNCTION IF EXISTS update_post_comments_count;
DROP FUNCTION IF EXISTS generate_ulid_as_uuid;
DROP FUNCTION IF EXISTS is_zero_uuid; 

DROP EXTENSION pgcrypto;
