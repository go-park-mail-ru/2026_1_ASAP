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

// Маппит int64 БД, которое может быть Null в *int64
func NullInt64ToPtrInt64(ni sql.NullInt64) *int64 {
	if ni.Valid {
		return &ni.Int64
	}
	return nil
}

// Маппит *int64 в число(int64) БД, которая может быть Null
func PtrInt64ToNullInt64(i *int64) sql.NullInt64 {
	if i != nil {
		return sql.NullInt64{Int64: *i, Valid: true}
	}
	return sql.NullInt64{}
}
