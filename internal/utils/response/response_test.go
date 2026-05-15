package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/dto/api"
	"github.com/stretchr/testify/require"
)

func TestPositiveResponse_Send(t *testing.T) {
	type fields struct{}

	type args struct {
		response interface{}
		status   int
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		args    args
		want    string
	}{
		{
			name:    "Sends JSON map with given status",
			prepare: nil,
			args: args{
				status:   http.StatusCreated,
				response: map[string]string{"key": "value"},
			},
			want: `"key":"value"`,
		},
		{
			name:    "Sends struct with 200",
			prepare: nil,
			args: args{
				status: http.StatusOK,
				response: struct {
					ID int `json:"id"`
				}{ID: 42},
			},
			want: `"id":42`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f fields
			if tt.prepare != nil {
				tt.prepare(&f)
			}

			rr := httptest.NewRecorder()
			Send(rr, tt.args.status, tt.args.response)

			require.Equal(t, tt.args.status, rr.Code)
			require.Equal(t, "application/json", rr.Header().Get("Content-Type"))
			require.Contains(t, rr.Body.String(), tt.want)
		})
	}
}

func TestNegativeResponse_Send(t *testing.T) {
	type fields struct{}

	type args struct {
		response interface{}
		status   int
	}

	tests := []struct {
		args           args
		prepare        func(*fields)
		name           string
		wantErrCode    api.ErrorCode
		wantErrMessage api.ErrorMessage
		wantStatus     int
	}{
		{
			name:    "Marshal failure returns 500 and internal error JSON",
			prepare: nil,
			args: args{
				status:   http.StatusOK,
				response: make(chan int),
			},
			wantStatus:     http.StatusInternalServerError,
			wantErrCode:    api.InternalError,
			wantErrMessage: api.InternalErrorMsg,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f fields
			if tt.prepare != nil {
				tt.prepare(&f)
			}

			rr := httptest.NewRecorder()
			Send(rr, tt.args.status, tt.args.response)

			require.Equal(t, tt.wantStatus, rr.Code)
			require.Equal(t, "application/json", rr.Header().Get("Content-Type"))

			var got api.ApiErrorResponse
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
			require.Equal(t, api.Error, got.Status)
			require.Len(t, got.Errors, 1)
			require.Equal(t, tt.wantErrCode, got.Errors[0].Code)
			require.Equal(t, tt.wantErrMessage, got.Errors[0].Message)
		})
	}
}
