package null

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func strPtr(s string) *string {
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func int64Ptr(v int64) *int64 {
	return &v
}

func TestPositiveNull_NullStringToPtrString(t *testing.T) {
	type fields struct{}

	type args struct {
		ns sql.NullString
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
		want    *string
	}{
		{
			name:    "Valid null string maps to pointer",
			prepare: nil,
			args:    args{ns: sql.NullString{String: "hello", Valid: true}},
			want:    strPtr("hello"),
		},
		{
			name:    "Invalid null string maps to nil",
			prepare: nil,
			args:    args{ns: sql.NullString{}},
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f fields
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			got := NullStringToPtrString(tt.args.ns)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestPositiveNull_StringPtrToNullString(t *testing.T) {
	type fields struct{}

	type args struct {
		s *string
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
		want    sql.NullString
	}{
		{
			name:    "Non-nil pointer is valid NullString",
			prepare: nil,
			args:    args{s: strPtr("x")},
			want:    sql.NullString{String: "x", Valid: true},
		},
		{
			name:    "Nil pointer is invalid NullString",
			prepare: nil,
			args:    args{s: nil},
			want:    sql.NullString{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f fields
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			got := StringPtrToNullString(tt.args.s)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestPositiveNull_NullTimeToPtrTime(t *testing.T) {
	type fields struct{}

	ts := time.Date(2026, 3, 15, 12, 30, 0, 0, time.UTC)

	type args struct {
		nt sql.NullTime
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
		want    *time.Time
	}{
		{
			name:    "Valid null time maps to pointer",
			prepare: nil,
			args:    args{nt: sql.NullTime{Time: ts, Valid: true}},
			want:    timePtr(ts),
		},
		{
			name:    "Invalid null time maps to nil",
			prepare: nil,
			args:    args{nt: sql.NullTime{}},
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f fields
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			got := NullTimeToPtrTime(tt.args.nt)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestPositiveNull_TimePtrToNullTime(t *testing.T) {
	type fields struct{}

	ts := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	type args struct {
		tm *time.Time
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
		want    sql.NullTime
	}{
		{
			name:    "Non-nil time is valid NullTime",
			prepare: nil,
			args:    args{tm: timePtr(ts)},
			want:    sql.NullTime{Time: ts, Valid: true},
		},
		{
			name:    "Nil time is invalid NullTime",
			prepare: nil,
			args:    args{tm: nil},
			want:    sql.NullTime{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f fields
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			got := TimePtrToNullTime(tt.args.tm)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestPositiveNull_NullInt64ToPtrInt64(t *testing.T) {
	type fields struct{}

	type args struct {
		ni sql.NullInt64
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
		want    *int64
	}{
		{
			name:    "Valid null int64 maps to pointer",
			prepare: nil,
			args:    args{ni: sql.NullInt64{Int64: 99, Valid: true}},
			want:    int64Ptr(99),
		},
		{
			name:    "Invalid null int64 maps to nil",
			prepare: nil,
			args:    args{ni: sql.NullInt64{}},
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f fields
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			got := NullInt64ToPtrInt64(tt.args.ni)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestPositiveNull_PtrInt64ToNullInt64(t *testing.T) {
	type fields struct{}

	type args struct {
		i *int64
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
		want    sql.NullInt64
	}{
		{
			name:    "Non-nil int64 is valid NullInt64",
			prepare: nil,
			args:    args{i: int64Ptr(-7)},
			want:    sql.NullInt64{Int64: -7, Valid: true},
		},
		{
			name:    "Nil int64 is invalid NullInt64",
			prepare: nil,
			args:    args{i: nil},
			want:    sql.NullInt64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f fields
			if tt.prepare != nil {
				tt.prepare(&f)
			}
			got := PtrInt64ToNullInt64(tt.args.i)
			require.Equal(t, tt.want, got)
		})
	}
}
