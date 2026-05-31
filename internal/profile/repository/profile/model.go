package profile

import (
	"database/sql"
)

type ProfileModel struct {
	BirthDate sql.NullTime
	LastSeen  sql.NullTime
	FirstName string
	LastName  sql.NullString
	Avatar    sql.NullString
	Bio       sql.NullString
	UserId    int64
}
