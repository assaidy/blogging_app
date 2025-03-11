package repo

import (
	"database/sql"
	"errors"
)

func IsNotFoundError(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

// NOTE: order is very important here.
// order follows kind's id in db.
const (
	NotificationKindNewFollower = iota + 1
	NotificationKindNewPost
)
