//go:generate mockgen -destination=mock/contacts_mock.go -package=mock github.com/go-park-mail-ru/2026_1_ASAP/gen/go/profile/v1 ProfileClient
package contacts

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	profilev1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/profile/v1"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/contacts/mock"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/middleware"
	contactdto "github.com/go-park-mail-ru/2026_1_ASAP/internal/profile/dto/contact"
)

func strPtr(s string) *string {
	return &s
}

func TestPositiveGatewayContactsHandler_GetContacts(t *testing.T) {
	type fields struct {
		profileClient *mock.MockProfileClient
	}

	type args struct {
		userID int64
	}

	//now := time.Now()

	tests := []struct {
		prepare func(*fields)
		want    int
		name    string
		args    args
	}{
		{
			name: "Successful get contacts",
			prepare: func(f *fields) {
				f.profileClient.EXPECT().ListContacts(gomock.Any(), &profilev1.RequestListContacts{
					UserId: 100,
				}).Return(&profilev1.ResponseListContacts{
					Contacts: []*profilev1.ContactItem{
						{
							UserId:           100,
							ContactUserId:    101,
							FirstName:        "John",
							LastName:         strPtr("Doe"),
							ContactAvatarUrl: strPtr("avatar.jpg"),
							CreatedAt:        timestamppb.Now(),
						},
						{
							UserId:           100,
							ContactUserId:    102,
							FirstName:        "Jane",
							LastName:         nil,
							ContactAvatarUrl: nil,
							CreatedAt:        timestamppb.Now(),
						},
					},
				}, nil)
			},
			args: args{userID: 100},
			want: http.StatusOK,
		},
		{
			name: "Get contacts empty list",
			prepare: func(f *fields) {
				f.profileClient.EXPECT().ListContacts(gomock.Any(), &profilev1.RequestListContacts{
					UserId: 200,
				}).Return(&profilev1.ResponseListContacts{
					Contacts: []*profilev1.ContactItem{},
				}, nil)
			},
			args: args{userID: 200},
			want: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				profileClient: mock.NewMockProfileClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayContactsHandler{
				Profile: f.profileClient,
			}

			r := chi.NewRouter()
			r.Get("/contacts", handler.GetContacts)

			req := httptest.NewRequest(http.MethodGet, "/contacts", nil)
			ctx := context.WithValue(req.Context(), middleware.UserID, tt.args.userID)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestNegativeGatewayContactsHandler_GetContacts(t *testing.T) {
	type fields struct {
		profileClient *mock.MockProfileClient
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		want    int
		userID  interface{}
	}{
		{
			name:   "Missing user_id in context",
			want:   http.StatusUnauthorized,
			userID: nil,
		},
		{
			name:   "Invalid user_id type",
			want:   http.StatusUnauthorized,
			userID: "invalid",
		},
		{
			name: "Profile service error",
			prepare: func(f *fields) {
				f.profileClient.EXPECT().ListContacts(gomock.Any(), &profilev1.RequestListContacts{
					UserId: 100,
				}).Return(nil, status.Error(codes.Internal, "internal error"))
			},
			want:   http.StatusInternalServerError,
			userID: int64(100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				profileClient: mock.NewMockProfileClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayContactsHandler{
				Profile: f.profileClient,
			}

			r := chi.NewRouter()
			r.Get("/contacts", handler.GetContacts)

			req := httptest.NewRequest(http.MethodGet, "/contacts", nil)
			if tt.userID != nil {
				if uid, ok := tt.userID.(int64); ok {
					ctx := context.WithValue(req.Context(), middleware.UserID, uid)
					req = req.WithContext(ctx)
				} else {
					ctx := context.WithValue(req.Context(), middleware.UserID, tt.userID)
					req = req.WithContext(ctx)
				}
			}
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestPositiveGatewayContactsHandler_CreateContact(t *testing.T) {
	type fields struct {
		profileClient *mock.MockProfileClient
	}

	type args struct {
		body   contactdto.AddContactRequest
		userID int64
	}

	//now := time.Now()

	tests := []struct {
		prepare func(*fields)
		want    int
		name    string
		args    args
	}{
		{
			name: "Successful create contact",
			prepare: func(f *fields) {
				f.profileClient.EXPECT().AddContact(gomock.Any(), &profilev1.RequestAddContact{
					UserId:        100,
					ContactUserId: 101,
					FirstName:     "John",
					LastName:      strPtr("Doe"),
				}).Return(&profilev1.ResponseAddContact{
					Contact: &profilev1.ContactItem{
						UserId:           100,
						ContactUserId:    101,
						FirstName:        "John",
						LastName:         strPtr("Doe"),
						ContactAvatarUrl: strPtr("avatar.jpg"),
						CreatedAt:        timestamppb.Now(),
					},
				}, nil)
			},
			args: args{
				userID: 100,
				body: contactdto.AddContactRequest{
					ContactUserID: 101,
					FirstName:     "John",
					LastName:      strPtr("Doe"),
				},
			},
			want: http.StatusCreated,
		},
		{
			name: "Successful create contact without last name",
			prepare: func(f *fields) {
				f.profileClient.EXPECT().AddContact(gomock.Any(), &profilev1.RequestAddContact{
					UserId:        100,
					ContactUserId: 102,
					FirstName:     "Jane",
					LastName:      nil,
				}).Return(&profilev1.ResponseAddContact{
					Contact: &profilev1.ContactItem{
						UserId:        100,
						ContactUserId: 102,
						FirstName:     "Jane",
						LastName:      nil,
						CreatedAt:     timestamppb.Now(),
					},
				}, nil)
			},
			args: args{
				userID: 100,
				body: contactdto.AddContactRequest{
					ContactUserID: 102,
					FirstName:     "Jane",
					LastName:      nil,
				},
			},
			want: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				profileClient: mock.NewMockProfileClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayContactsHandler{
				Profile: f.profileClient,
			}

			r := chi.NewRouter()
			r.Post("/contacts", handler.CreateContact)

			bodyBytes, _ := json.Marshal(tt.args.body)
			req := httptest.NewRequest(http.MethodPost, "/contacts", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.WithValue(req.Context(), middleware.UserID, tt.args.userID)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestNegativeGatewayContactsHandler_CreateContact(t *testing.T) {
	type fields struct {
		profileClient *mock.MockProfileClient
	}

	type args struct {
		body interface{}
	}

	tests := []struct {
		want    int
		prepare func(*fields)
		name    string
		args    args
	}{
		{
			name: "Invalid JSON body",
			args: args{
				body: "invalid json",
			},
			want: http.StatusBadRequest,
		},
		{
			name: "Missing user_id",
			args: args{
				body: contactdto.AddContactRequest{
					ContactUserID: 101,
					FirstName:     "John",
				},
			},
			want: http.StatusUnauthorized,
		},
		{
			name: "Invalid contact_user_id (zero)",
			args: args{
				body: contactdto.AddContactRequest{
					ContactUserID: 0,
					FirstName:     "John",
				},
			},
			want: http.StatusBadRequest,
		},
		{
			name: "Contact already exists",
			prepare: func(f *fields) {
				f.profileClient.EXPECT().AddContact(gomock.Any(), &profilev1.RequestAddContact{
					UserId:        100,
					ContactUserId: 101,
					FirstName:     "John",
					LastName:      nil,
				}).Return(nil, status.Error(codes.AlreadyExists, "contact already exists"))
			},
			args: args{
				body: contactdto.AddContactRequest{
					ContactUserID: 101,
					FirstName:     "John",
				},
			},
			want: http.StatusConflict,
		},
		{
			name: "User not found",
			prepare: func(f *fields) {
				f.profileClient.EXPECT().AddContact(gomock.Any(), &profilev1.RequestAddContact{
					UserId:        100,
					ContactUserId: 999,
					FirstName:     "John",
					LastName:      nil,
				}).Return(nil, status.Error(codes.NotFound, "user not found"))
			},
			args: args{
				body: contactdto.AddContactRequest{
					ContactUserID: 999,
					FirstName:     "John",
				},
			},
			want: http.StatusNotFound,
		},
		{
			name: "Contact with yourself",
			prepare: func(f *fields) {
				f.profileClient.EXPECT().AddContact(gomock.Any(), &profilev1.RequestAddContact{
					UserId:        100,
					ContactUserId: 100,
					FirstName:     "John",
					LastName:      nil,
				}).Return(nil, status.Error(codes.InvalidArgument, "cannot add yourself as contact"))
			},
			args: args{
				body: contactdto.AddContactRequest{
					ContactUserID: 100,
					FirstName:     "John",
				},
			},
			want: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				profileClient: mock.NewMockProfileClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayContactsHandler{
				Profile: f.profileClient,
			}

			r := chi.NewRouter()
			r.Post("/contacts", handler.CreateContact)

			var bodyBytes []byte
			if str, ok := tt.args.body.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tt.args.body)
			}

			req := httptest.NewRequest(http.MethodPost, "/contacts", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			// Добавляем user_id в контекст только если это не тест на missing user_id
			if tt.name != "Missing user_id" {
				ctx := context.WithValue(req.Context(), middleware.UserID, int64(100))
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestPositiveGatewayContactsHandler_DeleteContact(t *testing.T) {
	type fields struct {
		profileClient *mock.MockProfileClient
	}

	type args struct {
		contactUserID int64
		userID        int64
	}

	tests := []struct {
		prepare func(*fields)
		want    int
		name    string
		args    args
	}{
		{
			name: "Successful delete contact",
			prepare: func(f *fields) {
				f.profileClient.EXPECT().DeleteContact(gomock.Any(), &profilev1.RequestDeleteContact{
					UserId:        100,
					ContactUserId: 101,
				}).Return(&profilev1.ResponseDeleteContact{}, nil)
			},
			args: args{
				userID:        100,
				contactUserID: 101,
			},
			want: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				profileClient: mock.NewMockProfileClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayContactsHandler{
				Profile: f.profileClient,
			}

			r := chi.NewRouter()
			r.Delete("/contacts/{contact_user_id}", handler.DeleteContact)

			req := httptest.NewRequest(http.MethodDelete, "/contacts/101", nil)
			ctx := context.WithValue(req.Context(), middleware.UserID, tt.args.userID)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestNegativeGatewayContactsHandler_DeleteContact(t *testing.T) {
	type fields struct {
		profileClient *mock.MockProfileClient
	}

	tests := []struct {
		name          string
		prepare       func(*fields)
		want          int
		contactUserID string
		userID        interface{}
	}{
		{
			name:          "Invalid contact_user_id",
			want:          http.StatusBadRequest,
			contactUserID: "invalid",
			userID:        int64(100),
		},
		{
			name:          "Missing user_id",
			want:          http.StatusUnauthorized,
			contactUserID: "101",
			userID:        nil,
		},
		{
			name:          "Contact not found",
			contactUserID: "999",
			userID:        int64(100),
			prepare: func(f *fields) {
				f.profileClient.EXPECT().DeleteContact(gomock.Any(), &profilev1.RequestDeleteContact{
					UserId:        100,
					ContactUserId: 999,
				}).Return(nil, status.Error(codes.NotFound, "contact not found"))
			},
			want: http.StatusNotFound,
		},
		{
			name:          "Internal server error",
			contactUserID: "101",
			userID:        int64(100),
			prepare: func(f *fields) {
				f.profileClient.EXPECT().DeleteContact(gomock.Any(), &profilev1.RequestDeleteContact{
					UserId:        100,
					ContactUserId: 101,
				}).Return(nil, status.Error(codes.Internal, "internal error"))
			},
			want: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				profileClient: mock.NewMockProfileClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayContactsHandler{
				Profile: f.profileClient,
			}

			r := chi.NewRouter()
			r.Delete("/contacts/{contact_user_id}", handler.DeleteContact)

			req := httptest.NewRequest(http.MethodDelete, "/contacts/"+tt.contactUserID, nil)

			if tt.userID != nil {
				if uid, ok := tt.userID.(int64); ok {
					ctx := context.WithValue(req.Context(), middleware.UserID, uid)
					req = req.WithContext(ctx)
				}
			}

			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}
