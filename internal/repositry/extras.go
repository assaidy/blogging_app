package repositry

import (
	"database/sql"
	"errors"
)

func IsNotFoundError(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

const (
	NotificationKindNewFollower = iota + 1
	NotificationKindNewPost
)
