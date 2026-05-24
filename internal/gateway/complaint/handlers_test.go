//go:generate mockgen -destination=mock/complaint_mock.go -package=mock github.com/go-park-mail-ru/2026_1_ASAP/gen/go/complaint/v1 ComplaintClient
package complaint

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
	"google.golang.org/protobuf/types/known/timestamppb"

	complaintv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/complaint/v1"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/complaint/mock"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/gateway/middleware"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/grpcerr"
)

func ptrInt64(i int64) *int64 {
	return &i
}

func TestPositiveGatewayComplaintHandler_CreateComplaintUnAuthorized(t *testing.T) {
	type fields struct {
		complaintService *mock.MockComplaintClient
	}

	type args struct {
		body map[string]interface{}
	}

	tests := []struct {
		prepare func(*fields)
		want    int
		name    string
		args    args
	}{
		{
			name: "Successful create unauthorized complaint",
			prepare: func(f *fields) {
				f.complaintService.EXPECT().CreateComplaint(gomock.Any(), &complaintv1.RequestCreateComplaint{
					Type: complaintv1.ComplaintType_COMPLAINT_TYPE_BUG,
					Feedback: &complaintv1.Feedback{
						FeedbackName:  "John Doe",
						FeedbackEmail: "john@example.com",
					},
					Body:   "Bug description",
					File:   nil,
					UserId: nil,
				}).Return(&complaintv1.ResponseCreateComplaint{
					Complaint: &complaintv1.ComplaintItem{
						Id:     1,
						Type:   "bug",
						Status: "new",
						Feedback: &complaintv1.Feedback{
							FeedbackName:  "John Doe",
							FeedbackEmail: "john@example.com",
						},
						Body:      "Bug description",
						UserId:    0,
						CreatedAt: timestamppb.Now(),
						UpdatedAt: timestamppb.Now(),
					},
				}, nil)
			},
			args: args{
				body: map[string]interface{}{
					"type": "bug",
					"feedback": map[string]string{
						"feedback_name":  "John Doe",
						"feedback_email": "john@example.com",
					},
					"body": "Bug description",
				},
			},
			want: http.StatusOK,
		},
		{
			name: "Successful create complaint with suggestion type",
			prepare: func(f *fields) {
				f.complaintService.EXPECT().CreateComplaint(gomock.Any(), &complaintv1.RequestCreateComplaint{
					Type: complaintv1.ComplaintType_COMPLAINT_TYPE_PRODUCT,
					Feedback: &complaintv1.Feedback{
						FeedbackName:  "Jane Smith",
						FeedbackEmail: "jane@example.com",
					},
					Body:   "Product suggestion",
					File:   nil,
					UserId: nil,
				}).Return(&complaintv1.ResponseCreateComplaint{
					Complaint: &complaintv1.ComplaintItem{
						Id:     2,
						Type:   "product",
						Status: "new",
						Feedback: &complaintv1.Feedback{
							FeedbackName:  "Jane Smith",
							FeedbackEmail: "jane@example.com",
						},
						Body:      "Product suggestion",
						UserId:    0,
						CreatedAt: timestamppb.Now(),
						UpdatedAt: timestamppb.Now(),
					},
				}, nil)
			},
			args: args{
				body: map[string]interface{}{
					"type": "suggestion",
					"feedback": map[string]string{
						"feedback_name":  "Jane Smith",
						"feedback_email": "jane@example.com",
					},
					"body": "Product suggestion",
				},
			},
			want: http.StatusOK,
		},
		{
			name: "Successful create complaint with complaint type",
			prepare: func(f *fields) {
				f.complaintService.EXPECT().CreateComplaint(gomock.Any(), &complaintv1.RequestCreateComplaint{
					Type: complaintv1.ComplaintType_COMPLAINT_TYPE_UPGRADE,
					Feedback: &complaintv1.Feedback{
						FeedbackName:  "Bob Wilson",
						FeedbackEmail: "bob@example.com",
					},
					Body:   "Upgrade request",
					File:   nil,
					UserId: nil,
				}).Return(&complaintv1.ResponseCreateComplaint{
					Complaint: &complaintv1.ComplaintItem{
						Id:     3,
						Type:   "upgrade",
						Status: "new",
						Feedback: &complaintv1.Feedback{
							FeedbackName:  "Bob Wilson",
							FeedbackEmail: "bob@example.com",
						},
						Body:      "Upgrade request",
						UserId:    0,
						CreatedAt: timestamppb.Now(),
						UpdatedAt: timestamppb.Now(),
					},
				}, nil)
			},
			args: args{
				body: map[string]interface{}{
					"type": "complaint",
					"feedback": map[string]string{
						"feedback_name":  "Bob Wilson",
						"feedback_email": "bob@example.com",
					},
					"body": "Upgrade request",
				},
			},
			want: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				complaintService: mock.NewMockComplaintClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayComplaintHandler{
				ComplaintService: f.complaintService,
			}

			r := chi.NewRouter()
			r.Post("/complaints", handler.CreateComplaintUnAuthorized)

			bodyBytes, _ := json.Marshal(tt.args.body)
			req := httptest.NewRequest(http.MethodPost, "/complaints", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestPositiveGatewayComplaintHandler_CreateComplaintAuthorized(t *testing.T) {
	type fields struct {
		complaintService *mock.MockComplaintClient
	}

	type args struct {
		body   map[string]interface{}
		userID int64
	}

	tests := []struct {
		prepare func(*fields)
		want    int
		name    string
		args    args
	}{
		{
			name: "Successful create authorized complaint",
			prepare: func(f *fields) {
				f.complaintService.EXPECT().CreateComplaint(gomock.Any(), &complaintv1.RequestCreateComplaint{
					Type: complaintv1.ComplaintType_COMPLAINT_TYPE_BUG,
					Feedback: &complaintv1.Feedback{
						FeedbackName:  "John Doe",
						FeedbackEmail: "john@example.com",
					},
					Body:   "Bug description",
					File:   nil,
					UserId: ptrInt64(100),
				}).Return(&complaintv1.ResponseCreateComplaint{
					Complaint: &complaintv1.ComplaintItem{
						Id:     1,
						Type:   "bug",
						Status: "new",
						Feedback: &complaintv1.Feedback{
							FeedbackName:  "John Doe",
							FeedbackEmail: "john@example.com",
						},
						Body:      "Bug description",
						UserId:    100,
						CreatedAt: timestamppb.Now(),
						UpdatedAt: timestamppb.Now(),
					},
				}, nil)
			},
			args: args{
				userID: 100,
				body: map[string]interface{}{
					"type": "bug",
					"feedback": map[string]string{
						"feedback_name":  "John Doe",
						"feedback_email": "john@example.com",
					},
					"body": "Bug description",
				},
			},
			want: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				complaintService: mock.NewMockComplaintClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayComplaintHandler{
				ComplaintService: f.complaintService,
			}

			r := chi.NewRouter()
			r.Post("/complaints/auth", handler.CreateComplaintAuthorized)

			bodyBytes, _ := json.Marshal(tt.args.body)
			req := httptest.NewRequest(http.MethodPost, "/complaints/auth", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.WithValue(req.Context(), middleware.UserID, tt.args.userID)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestNegativeGatewayComplaintHandler_CreateComplaint(t *testing.T) {
	type fields struct {
		complaintService *mock.MockComplaintClient
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
			name: "Missing type field - sends default unspecified type",
			prepare: func(f *fields) {
				f.complaintService.EXPECT().CreateComplaint(gomock.Any(), gomock.Any()).Return(&complaintv1.ResponseCreateComplaint{
					Complaint: &complaintv1.ComplaintItem{Id: 1},
				}, nil)
			},
			args: args{
				body: map[string]interface{}{
					"feedback": map[string]string{
						"feedback_name":  "John Doe",
						"feedback_email": "john@example.com",
					},
					"body": "Bug description",
				},
			},
			want: http.StatusOK, // Хендлер отправляет unspecified тип, но gR Call может пройти
		},
		{
			name: "Invalid complaint type - uses default unspecified",
			prepare: func(f *fields) {
				f.complaintService.EXPECT().CreateComplaint(gomock.Any(), gomock.Any()).Return(&complaintv1.ResponseCreateComplaint{
					Complaint: &complaintv1.ComplaintItem{Id: 1},
				}, nil)
			},
			args: args{
				body: map[string]interface{}{
					"type": "invalid",
					"feedback": map[string]string{
						"feedback_name":  "John Doe",
						"feedback_email": "john@example.com",
					},
					"body": "Bug description",
				},
			},
			want: http.StatusOK, // Хендлер конвертирует invalid в COMPLAINT_TYPE_UNSPECIFIED
		},
		{
			name: "Missing body field - sends empty body",
			prepare: func(f *fields) {
				f.complaintService.EXPECT().CreateComplaint(gomock.Any(), &complaintv1.RequestCreateComplaint{
					Type: complaintv1.ComplaintType_COMPLAINT_TYPE_BUG,
					Feedback: &complaintv1.Feedback{
						FeedbackName:  "John Doe",
						FeedbackEmail: "john@example.com",
					},
					Body:   "",
					File:   nil,
					UserId: nil,
				}).Return(&complaintv1.ResponseCreateComplaint{
					Complaint: &complaintv1.ComplaintItem{Id: 1},
				}, nil)
			},
			args: args{
				body: map[string]interface{}{
					"type": "bug",
					"feedback": map[string]string{
						"feedback_name":  "John Doe",
						"feedback_email": "john@example.com",
					},
				},
			},
			want: http.StatusOK,
		},
		{
			name: "Complaint not found error",
			prepare: func(f *fields) {
				f.complaintService.EXPECT().CreateComplaint(gomock.Any(), gomock.Any()).Return(nil, grpcerr.New(codes.NotFound, int32(complaintv1.ComplaintErrorCode_COMPLAINT_ERROR_NOT_FOUND), "not found"))
			},
			args: args{
				body: map[string]interface{}{
					"type": "bug",
					"feedback": map[string]string{
						"feedback_name":  "John Doe",
						"feedback_email": "john@example.com",
					},
					"body": "Bug description",
				},
			},
			want: http.StatusNotFound,
		},
		{
			name: "Invalid input error",
			prepare: func(f *fields) {
				f.complaintService.EXPECT().CreateComplaint(gomock.Any(), gomock.Any()).Return(nil, grpcerr.New(codes.InvalidArgument, int32(complaintv1.ComplaintErrorCode_COMPLAINT_ERROR_INVALID_INPUT), "invalid input"))
			},
			args: args{
				body: map[string]interface{}{
					"type": "bug",
					"feedback": map[string]string{
						"feedback_name":  "John Doe",
						"feedback_email": "john@example.com",
					},
					"body": "Bug description",
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
				complaintService: mock.NewMockComplaintClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayComplaintHandler{
				ComplaintService: f.complaintService,
			}

			r := chi.NewRouter()
			r.Post("/complaints", handler.CreateComplaintUnAuthorized)

			var bodyBytes []byte
			if str, ok := tt.args.body.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tt.args.body)
			}

			req := httptest.NewRequest(http.MethodPost, "/complaints", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestPositiveGatewayComplaintHandler_GetComplaint(t *testing.T) {
	type fields struct {
		complaintService *mock.MockComplaintClient
	}

	tests := []struct {
		prepare     func(*fields)
		want        int
		name        string
		complaintID string
	}{
		{
			name: "Successful get complaint by id",
			prepare: func(f *fields) {
				f.complaintService.EXPECT().GetComplaint(gomock.Any(), &complaintv1.RequestGetComplaint{
					ComplaintId: 1,
				}).Return(&complaintv1.ResponseGetComplaint{
					Complaint: &complaintv1.ComplaintItem{
						Id:     1,
						Type:   "bug",
						Status: "new",
						Feedback: &complaintv1.Feedback{
							FeedbackName:  "John Doe",
							FeedbackEmail: "john@example.com",
						},
						Body:      "Bug description",
						UserId:    100,
						CreatedAt: timestamppb.Now(),
						UpdatedAt: timestamppb.Now(),
					},
				}, nil)
			},
			complaintID: "1",
			want:        http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				complaintService: mock.NewMockComplaintClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayComplaintHandler{
				ComplaintService: f.complaintService,
			}

			r := chi.NewRouter()
			r.Get("/complaints/{id}", handler.GetComplaint)

			req := httptest.NewRequest(http.MethodGet, "/complaints/"+tt.complaintID, nil)
			ctx := context.WithValue(req.Context(), middleware.UserID, int64(100))
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestNegativeGatewayComplaintHandler_GetComplaint(t *testing.T) {
	type fields struct {
		complaintService *mock.MockComplaintClient
	}

	tests := []struct {
		name        string
		prepare     func(*fields)
		want        int
		complaintID string
		userID      interface{}
	}{
		{
			name:        "Invalid complaint id",
			want:        http.StatusBadRequest,
			complaintID: "invalid",
			userID:      int64(100),
		},
		{
			name:        "Missing user_id",
			want:        http.StatusUnauthorized,
			complaintID: "1",
			userID:      nil,
		},
		{
			name: "Complaint not found",
			prepare: func(f *fields) {
				f.complaintService.EXPECT().GetComplaint(gomock.Any(), &complaintv1.RequestGetComplaint{
					ComplaintId: 999,
				}).Return(nil, grpcerr.New(codes.NotFound, int32(complaintv1.ComplaintErrorCode_COMPLAINT_ERROR_NOT_FOUND), "not found"))
			},
			want:        http.StatusNotFound,
			complaintID: "999",
			userID:      int64(100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				complaintService: mock.NewMockComplaintClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayComplaintHandler{
				ComplaintService: f.complaintService,
			}

			r := chi.NewRouter()
			r.Get("/complaints/{id}", handler.GetComplaint)

			req := httptest.NewRequest(http.MethodGet, "/complaints/"+tt.complaintID, nil)
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

func TestPositiveGatewayComplaintHandler_GetMyComplaints(t *testing.T) {
	type fields struct {
		complaintService *mock.MockComplaintClient
	}

	tests := []struct {
		prepare func(*fields)
		want    int
		name    string
		userID  int64
	}{
		{
			name: "Successful get my complaints",
			prepare: func(f *fields) {
				f.complaintService.EXPECT().GetComplaintsByUser(gomock.Any(), &complaintv1.RequestGetComplaintsByUser{
					UserId: 100,
				}).Return(&complaintv1.ResponseGetComplaints{
					Complaints: []*complaintv1.ComplaintItem{
						{
							Id:     1,
							Type:   "bug",
							Status: "new",
							Feedback: &complaintv1.Feedback{
								FeedbackName:  "John Doe",
								FeedbackEmail: "john@example.com",
							},
							Body:      "Bug description",
							UserId:    100,
							CreatedAt: timestamppb.Now(),
							UpdatedAt: timestamppb.Now(),
						},
					},
				}, nil)
			},
			userID: 100,
			want:   http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				complaintService: mock.NewMockComplaintClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayComplaintHandler{
				ComplaintService: f.complaintService,
			}

			r := chi.NewRouter()
			r.Get("/complaints/my", handler.GetMyComplaints)

			req := httptest.NewRequest(http.MethodGet, "/complaints/my", nil)
			ctx := context.WithValue(req.Context(), middleware.UserID, tt.userID)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestNegativeGatewayComplaintHandler_GetMyComplaints(t *testing.T) {
	type fields struct {
		complaintService *mock.MockComplaintClient
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		want    int
		userID  interface{}
	}{
		{
			name:   "Missing user_id",
			want:   http.StatusUnauthorized,
			userID: nil,
		},
		{
			name: "Service error",
			prepare: func(f *fields) {
				f.complaintService.EXPECT().GetComplaintsByUser(gomock.Any(), &complaintv1.RequestGetComplaintsByUser{
					UserId: 100,
				}).Return(nil, grpcerr.New(codes.Internal, int32(complaintv1.ComplaintErrorCode_COMPLAINT_ERROR_INTERNAL), "internal error"))
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
				complaintService: mock.NewMockComplaintClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayComplaintHandler{
				ComplaintService: f.complaintService,
			}

			r := chi.NewRouter()
			r.Get("/complaints/my", handler.GetMyComplaints)

			req := httptest.NewRequest(http.MethodGet, "/complaints/my", nil)
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

func TestPositiveGatewayComplaintHandler_GetAllComplaints(t *testing.T) {
	type fields struct {
		complaintService *mock.MockComplaintClient
	}

	tests := []struct {
		prepare func(*fields)
		want    int
		name    string
	}{
		{
			name: "Successful get all complaints",
			prepare: func(f *fields) {
				f.complaintService.EXPECT().GetAllComplaints(gomock.Any(), &complaintv1.RequestGetAllComplaints{}).Return(&complaintv1.ResponseGetComplaints{
					Complaints: []*complaintv1.ComplaintItem{
						{
							Id:     1,
							Type:   "bug",
							Status: "new",
							Feedback: &complaintv1.Feedback{
								FeedbackName:  "John Doe",
								FeedbackEmail: "john@example.com",
							},
							Body:      "Bug description",
							UserId:    100,
							CreatedAt: timestamppb.Now(),
							UpdatedAt: timestamppb.Now(),
						},
					},
				}, nil)
			},
			want: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				complaintService: mock.NewMockComplaintClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayComplaintHandler{
				ComplaintService: f.complaintService,
			}

			r := chi.NewRouter()
			r.Get("/complaints/all", handler.GetAllComplaints)

			req := httptest.NewRequest(http.MethodGet, "/complaints/all", nil)
			ctx := context.WithValue(req.Context(), middleware.UserID, int64(100))
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestNegativeGatewayComplaintHandler_GetAllComplaints(t *testing.T) {
	type fields struct {
		complaintService *mock.MockComplaintClient
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		want    int
		userID  interface{}
	}{
		{
			name:   "Missing user_id",
			want:   http.StatusUnauthorized,
			userID: nil,
		},
		{
			name: "Service error",
			prepare: func(f *fields) {
				f.complaintService.EXPECT().GetAllComplaints(gomock.Any(), &complaintv1.RequestGetAllComplaints{}).Return(nil, grpcerr.New(codes.Internal, int32(complaintv1.ComplaintErrorCode_COMPLAINT_ERROR_INTERNAL), "internal error"))
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
				complaintService: mock.NewMockComplaintClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayComplaintHandler{
				ComplaintService: f.complaintService,
			}

			r := chi.NewRouter()
			r.Get("/complaints/all", handler.GetAllComplaints)

			req := httptest.NewRequest(http.MethodGet, "/complaints/all", nil)
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

func TestPositiveGatewayComplaintHandler_UpdateComplaintStatus(t *testing.T) {
	type fields struct {
		complaintService *mock.MockComplaintClient
	}

	type args struct {
		body map[string]interface{}
	}

	tests := []struct {
		prepare func(*fields)
		want    int
		name    string
		args    args
	}{
		{
			name: "Successful update complaint status to in_progress",
			prepare: func(f *fields) {
				f.complaintService.EXPECT().UpdateComplaintStatus(gomock.Any(), &complaintv1.RequestUpdateComplaintStatus{
					ComplaintId: 1,
					Status:      complaintv1.ComplaintStatus_COMPLAINT_STATUS_IN_PROGRESS,
				}).Return(&complaintv1.ResponseGetComplaint{
					Complaint: &complaintv1.ComplaintItem{
						Id:     1,
						Type:   "bug",
						Status: "in_progress",
						Feedback: &complaintv1.Feedback{
							FeedbackName:  "John Doe",
							FeedbackEmail: "john@example.com",
						},
						Body:      "Bug description",
						UserId:    100,
						CreatedAt: timestamppb.Now(),
						UpdatedAt: timestamppb.Now(),
					},
				}, nil)
			},
			args: args{
				body: map[string]interface{}{
					"complaint_id": float64(1),
					"status":       "in_progress",
				},
			},
			want: http.StatusOK,
		},
		{
			name: "Successful update complaint status to closed",
			prepare: func(f *fields) {
				f.complaintService.EXPECT().UpdateComplaintStatus(gomock.Any(), &complaintv1.RequestUpdateComplaintStatus{
					ComplaintId: 2,
					Status:      complaintv1.ComplaintStatus_COMPLAINT_STATUS_CLOSED,
				}).Return(&complaintv1.ResponseGetComplaint{
					Complaint: &complaintv1.ComplaintItem{
						Id:     2,
						Type:   "bug",
						Status: "closed",
						Feedback: &complaintv1.Feedback{
							FeedbackName:  "John Doe",
							FeedbackEmail: "john@example.com",
						},
						Body:      "Bug description",
						UserId:    100,
						CreatedAt: timestamppb.Now(),
						UpdatedAt: timestamppb.Now(),
					},
				}, nil)
			},
			args: args{
				body: map[string]interface{}{
					"complaint_id": float64(2),
					"status":       "closed",
				},
			},
			want: http.StatusOK,
		},
		{
			name: "Successful update complaint status to new",
			prepare: func(f *fields) {
				f.complaintService.EXPECT().UpdateComplaintStatus(gomock.Any(), &complaintv1.RequestUpdateComplaintStatus{
					ComplaintId: 3,
					Status:      complaintv1.ComplaintStatus_COMPLAINT_STATUS_NEW,
				}).Return(&complaintv1.ResponseGetComplaint{
					Complaint: &complaintv1.ComplaintItem{
						Id:     3,
						Type:   "bug",
						Status: "new",
						Feedback: &complaintv1.Feedback{
							FeedbackName:  "John Doe",
							FeedbackEmail: "john@example.com",
						},
						Body:      "Bug description",
						UserId:    100,
						CreatedAt: timestamppb.Now(),
						UpdatedAt: timestamppb.Now(),
					},
				}, nil)
			},
			args: args{
				body: map[string]interface{}{
					"complaint_id": float64(3),
					"status":       "new",
				},
			},
			want: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				complaintService: mock.NewMockComplaintClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayComplaintHandler{
				ComplaintService: f.complaintService,
			}

			r := chi.NewRouter()
			r.Put("/complaints/status", handler.UpdateComplaintStatus)

			bodyBytes, _ := json.Marshal(tt.args.body)
			req := httptest.NewRequest(http.MethodPut, "/complaints/status", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.WithValue(req.Context(), middleware.UserID, int64(100))
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestNegativeGatewayComplaintHandler_UpdateComplaintStatus(t *testing.T) {
	type fields struct {
		complaintService *mock.MockComplaintClient
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
			name: "Missing complaint_id - validation fails",
			args: args{
				body: map[string]interface{}{
					"status": "in_progress",
				},
			},
			want: http.StatusBadRequest,
		},
		{
			name: "Missing status - sends empty status",
			prepare: func(f *fields) {
				f.complaintService.EXPECT().UpdateComplaintStatus(gomock.Any(), &complaintv1.RequestUpdateComplaintStatus{
					ComplaintId: 1,
					Status:      complaintv1.ComplaintStatus_COMPLAINT_STATUS_UNSPECIFIED,
				}).Return(&complaintv1.ResponseGetComplaint{
					Complaint: &complaintv1.ComplaintItem{Id: 1},
				}, nil)
			},
			args: args{
				body: map[string]interface{}{
					"complaint_id": float64(1),
				},
			},
			want: http.StatusOK, // Хендлер отправляет unspecified статус
		},
		{
			name: "Invalid status value - sends unspecified",
			prepare: func(f *fields) {
				f.complaintService.EXPECT().UpdateComplaintStatus(gomock.Any(), &complaintv1.RequestUpdateComplaintStatus{
					ComplaintId: 1,
					Status:      complaintv1.ComplaintStatus_COMPLAINT_STATUS_UNSPECIFIED,
				}).Return(&complaintv1.ResponseGetComplaint{
					Complaint: &complaintv1.ComplaintItem{Id: 1},
				}, nil)
			},
			args: args{
				body: map[string]interface{}{
					"complaint_id": float64(1),
					"status":       "invalid",
				},
			},
			want: http.StatusOK,
		},
		{
			name: "Complaint not found",
			prepare: func(f *fields) {
				f.complaintService.EXPECT().UpdateComplaintStatus(gomock.Any(), &complaintv1.RequestUpdateComplaintStatus{
					ComplaintId: 999,
					Status:      complaintv1.ComplaintStatus_COMPLAINT_STATUS_IN_PROGRESS,
				}).Return(nil, grpcerr.New(codes.NotFound, int32(complaintv1.ComplaintErrorCode_COMPLAINT_ERROR_NOT_FOUND), "not found"))
			},
			args: args{
				body: map[string]interface{}{
					"complaint_id": float64(999),
					"status":       "in_progress",
				},
			},
			want: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				complaintService: mock.NewMockComplaintClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayComplaintHandler{
				ComplaintService: f.complaintService,
			}

			r := chi.NewRouter()
			r.Put("/complaints/status", handler.UpdateComplaintStatus)

			var bodyBytes []byte
			if str, ok := tt.args.body.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, _ = json.Marshal(tt.args.body)
			}

			req := httptest.NewRequest(http.MethodPut, "/complaints/status", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.WithValue(req.Context(), middleware.UserID, int64(100))
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestPositiveGatewayAnalyticHandler_GetUserComplaintAnalytic(t *testing.T) {
	type fields struct {
		complaintService *mock.MockComplaintClient
	}

	tests := []struct {
		prepare func(*fields)
		want    int
		name    string
		userID  int64
	}{
		{
			name: "Successful get user complaint analytic",
			prepare: func(f *fields) {
				f.complaintService.EXPECT().GetUserComplaintAnalytic(gomock.Any(), &complaintv1.RequestGetUserComplaintAnalytic{
					UserId: 100,
				}).Return(&complaintv1.ResponseGetUserComplaintAnalytic{
					CountStatus: &complaintv1.CountStatus{
						CountStatusOpened: 5,
						CountStatusInWork: 3,
						CountStatusClosed: 2,
					},
					CountType: &complaintv1.CountType{
						CountBug:     4,
						CountUpgrade: 3,
						CountProduct: 3,
					},
				}, nil)
			},
			userID: 100,
			want:   http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				complaintService: mock.NewMockComplaintClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayAnalyticHandler{
				ComplaintService: f.complaintService,
			}

			r := chi.NewRouter()
			r.Get("/complaints/analytic", handler.GetUserComplaintAnalytic)

			req := httptest.NewRequest(http.MethodGet, "/complaints/analytic", nil)
			ctx := context.WithValue(req.Context(), middleware.UserID, tt.userID)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code)
		})
	}
}

func TestNegativeGatewayAnalyticHandler_GetUserComplaintAnalytic(t *testing.T) {
	type fields struct {
		complaintService *mock.MockComplaintClient
	}

	tests := []struct {
		name    string
		prepare func(*fields)
		want    int
		userID  interface{}
	}{
		{
			name:   "Missing user_id",
			want:   http.StatusUnauthorized,
			userID: nil,
		},
		{
			name: "Service error",
			prepare: func(f *fields) {
				f.complaintService.EXPECT().GetUserComplaintAnalytic(gomock.Any(), &complaintv1.RequestGetUserComplaintAnalytic{
					UserId: 100,
				}).Return(nil, grpcerr.New(codes.Internal, int32(complaintv1.ComplaintErrorCode_COMPLAINT_ERROR_INTERNAL), "internal error"))
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
				complaintService: mock.NewMockComplaintClient(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			handler := &GatewayAnalyticHandler{
				ComplaintService: f.complaintService,
			}

			r := chi.NewRouter()
			r.Get("/complaints/analytic", handler.GetUserComplaintAnalytic)

			req := httptest.NewRequest(http.MethodGet, "/complaints/analytic", nil)
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
