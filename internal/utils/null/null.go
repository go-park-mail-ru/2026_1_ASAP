package null

import (
	"database/sql"
	"time"
)

// Маппит строку БД, которое может быть Null в *string
func NullStringToPtrString(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

// Маппит тип времени БД, который может быть Null в *time.Time
func NullTimeToPtrTime(nt sql.NullTime) *time.Time {
	if nt.Valid {
		return &nt.Time
	}
	return nil
}

// Маппит *string в строку БД, которая может быть Null
func StringPtrToNullString(s *string) sql.NullString {
	if s != nil {
		return sql.NullString{String: *s, Valid: true}
	}
	return sql.NullString{}
}

// Маппит *Time в тип время БД, который может быть Null
func TimePtrToNullTime(t *time.Time) sql.NullTime {
	if t != nil {
		return sql.NullTime{Time: *t, Valid: true}
	}
	return sql.NullTime{}
}
