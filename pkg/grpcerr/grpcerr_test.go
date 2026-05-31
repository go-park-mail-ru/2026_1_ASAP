package grpcerr

import (
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewAndError(t *testing.T) {
	tests := []struct {
		err      error
		name     string
		wantCode codes.Code
		wantApp  int32
		wantMsg  string
	}{
		{
			name:     "with app details",
			err:      New(codes.NotFound, 42, "missing"),
			wantCode: codes.NotFound,
			wantApp:  42,
			wantMsg:  "missing",
		},
		{
			name:     "plain grpc status",
			err:      status.Error(codes.PermissionDenied, "denied"),
			wantCode: codes.PermissionDenied,
			wantApp:  0,
			wantMsg:  "denied",
		},
		{
			name:     "plain error",
			err:      errors.New("boom"),
			wantCode: codes.Internal,
			wantApp:  0,
			wantMsg:  "boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCode, gotApp, gotMsg := Error(tt.err)
			if gotCode != tt.wantCode || gotApp != tt.wantApp || gotMsg != tt.wantMsg {
				t.Fatalf("Error() = (%v, %d, %q), want (%v, %d, %q)", gotCode, gotApp, gotMsg, tt.wantCode, tt.wantApp, tt.wantMsg)
			}
		})
	}
}
