//go:generate mockgen -destination=mock/complaint_usecase_mock.go -package=mock github.com/go-park-mail-ru/2026_1_ASAP/internal/complaint/transport/grpc ComplaintUsecase
//go:generate mockgen -destination=mock/analytic_usecase_mock.go -package=mock github.com/go-park-mail-ru/2026_1_ASAP/internal/complaint/transport/grpc AnalyticUsecase
package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	complaintv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/complaint/v1"
	domainComplaint "github.com/go-park-mail-ru/2026_1_ASAP/internal/complaint/domain/complaint"
	dtoAnalytic "github.com/go-park-mail-ru/2026_1_ASAP/internal/complaint/dto/analytic"
	dtoComplaint "github.com/go-park-mail-ru/2026_1_ASAP/internal/complaint/dto/complaint"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/complaint/transport/grpc/mock"
)

func strPtr(s string) *string {
	return &s
}

func TestPositiveComplaintServer_CreateComplaint_Unauthorized(t *testing.T) {
	type fields struct {
		complaintUsecase *mock.MockComplaintUsecase
		analyticUsecase  *mock.MockAnalyticUsecase
	}

	type args struct {
		ctx context.Context
		req *complaintv1.RequestCreateComplaint
	}

	now := time.Now()

	tests := []struct {
		prepare func(*fields)
		want    *complaintv1.ResponseCreateComplaint
		name    string
		args    args
	}{
		{
			name: "Successful create unauthorized complaint",
			prepare: func(f *fields) {
				f.complaintUsecase.EXPECT().CreateUnAuthrozied(gomock.Any(), dtoComplaint.RequestCreateComplaint{
					Type: "bug",
					FeedBackInfo: dtoComplaint.FeedbackDTO{
						FeedBackName:  "John Doe",
						FeedBackEmail: "john@example.com",
					},
					Body: "Bug description",
					File: nil,
				}).Return(&dtoComplaint.ResponseCreateComplaint{
					ComplaintDTO: dtoComplaint.ComplaintDTO{
						ID:     1,
						Type:   "bug",
						Status: "new",
						FeedbackDTO: dtoComplaint.FeedbackDTO{
							FeedBackName:  "John Doe",
							FeedBackEmail: "john@example.com",
						},
						Body:      "Bug description",
						UserID:    0,
						CreatedAt: now,
						UpdatedAt: now,
					},
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &complaintv1.RequestCreateComplaint{
					Type: complaintv1.ComplaintType_COMPLAINT_TYPE_BUG,
					Feedback: &complaintv1.Feedback{
						FeedbackName:  "John Doe",
						FeedbackEmail: "john@example.com",
					},
					Body:   "Bug description",
					File:   nil,
					UserId: nil,
				},
			},
			want: &complaintv1.ResponseCreateComplaint{
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
					CreatedAt: timestamppb.New(now),
					UpdatedAt: timestamppb.New(now),
				},
			},
		},
		{
			name: "Successful create unauthorized complaint with file",
			prepare: func(f *fields) {
				f.complaintUsecase.EXPECT().CreateUnAuthrozied(gomock.Any(), gomock.Any()).Return(&dtoComplaint.ResponseCreateComplaint{
					ComplaintDTO: dtoComplaint.ComplaintDTO{
						ID:     2,
						Type:   "bug",
						Status: "new",
						FeedbackDTO: dtoComplaint.FeedbackDTO{
							FeedBackName:  "Jane Smith",
							FeedBackEmail: "jane@example.com",
						},
						Body:         "Bug with attachment",
						UserID:       0,
						AttachmentURL: strPtr("attachment.jpg"),
						CreatedAt:    now,
						UpdatedAt:    now,
					},
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &complaintv1.RequestCreateComplaint{
					Type: complaintv1.ComplaintType_COMPLAINT_TYPE_BUG,
					Feedback: &complaintv1.Feedback{
						FeedbackName:  "Jane Smith",
						FeedbackEmail: "jane@example.com",
					},
					Body: "Bug with attachment",
					File: &complaintv1.ComplaintFile{
						Content: []byte("file content"),
						Type:    "image/jpeg",
					},
					UserId: nil,
				},
			},
			want: &complaintv1.ResponseCreateComplaint{
				Complaint: &complaintv1.ComplaintItem{
					Id:     2,
					Type:   "bug",
					Status: "new",
					Feedback: &complaintv1.Feedback{
						FeedbackName:  "Jane Smith",
						FeedbackEmail: "jane@example.com",
					},
					Body:         "Bug with attachment",
					UserId:       0,
					AttachmentUrl: strPtr("attachment.jpg"),
					CreatedAt:    timestamppb.New(now),
					UpdatedAt:    timestamppb.New(now),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				complaintUsecase: mock.NewMockComplaintUsecase(ctrl),
				analyticUsecase:  mock.NewMockAnalyticUsecase(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ComplaintServer{
				complaintUsecase: f.complaintUsecase,
				analyticUsecase:  f.analyticUsecase,
				logger:           zap.NewNop(),
			}

			resp, err := s.CreateComplaint(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want.GetComplaint().GetId(), resp.GetComplaint().GetId())
			require.Equal(t, tt.want.GetComplaint().GetType(), resp.GetComplaint().GetType())
		})
	}
}

func TestPositiveComplaintServer_CreateComplaint_Authorized(t *testing.T) {
	type fields struct {
		complaintUsecase *mock.MockComplaintUsecase
		analyticUsecase  *mock.MockAnalyticUsecase
	}

	type args struct {
		ctx context.Context
		req *complaintv1.RequestCreateComplaint
	}

	now := time.Now()

	tests := []struct {
		prepare func(*fields)
		want    *complaintv1.ResponseCreateComplaint
		name    string
		args    args
	}{
		{
			name: "Successful create authorized complaint",
			prepare: func(f *fields) {
				//userId := int64(100)
				f.complaintUsecase.EXPECT().CreateAuthrozied(gomock.Any(), int64(100), dtoComplaint.RequestCreateComplaint{
					Type: "bug",
					FeedBackInfo: dtoComplaint.FeedbackDTO{
						FeedBackName:  "John Doe",
						FeedBackEmail: "john@example.com",
					},
					Body: "Bug description",
					File: nil,
				}).Return(&dtoComplaint.ResponseCreateComplaint{
					ComplaintDTO: dtoComplaint.ComplaintDTO{
						ID:     1,
						Type:   "bug",
						Status: "new",
						FeedbackDTO: dtoComplaint.FeedbackDTO{
							FeedBackName:  "John Doe",
							FeedBackEmail: "john@example.com",
						},
						Body:      "Bug description",
						UserID:    100,
						CreatedAt: now,
						UpdatedAt: now,
					},
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &complaintv1.RequestCreateComplaint{
					Type: complaintv1.ComplaintType_COMPLAINT_TYPE_BUG,
					Feedback: &complaintv1.Feedback{
						FeedbackName:  "John Doe",
						FeedbackEmail: "john@example.com",
					},
					Body:   "Bug description",
					File:   nil,
					UserId: ptrInt64(100),
				},
			},
			want: &complaintv1.ResponseCreateComplaint{
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
					CreatedAt: timestamppb.New(now),
					UpdatedAt: timestamppb.New(now),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				complaintUsecase: mock.NewMockComplaintUsecase(ctrl),
				analyticUsecase:  mock.NewMockAnalyticUsecase(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ComplaintServer{
				complaintUsecase: f.complaintUsecase,
				analyticUsecase:  f.analyticUsecase,
				logger:           zap.NewNop(),
			}

			resp, err := s.CreateComplaint(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want.GetComplaint().GetId(), resp.GetComplaint().GetId())
		})
	}
}

func TestNegativeComplaintServer_CreateComplaint(t *testing.T) {
	type fields struct {
		complaintUsecase *mock.MockComplaintUsecase
		analyticUsecase  *mock.MockAnalyticUsecase
	}

	type args struct {
		ctx context.Context
		req *complaintv1.RequestCreateComplaint
	}

	tests := []struct {
		wantCode codes.Code
		prepare  func(*fields)
		name     string
		args     args
	}{
		{
			name: "Nil request",
			args: args{
				ctx: context.Background(),
				req: nil,
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Unspecified type",
			args: args{
				ctx: context.Background(),
				req: &complaintv1.RequestCreateComplaint{
					Type:     complaintv1.ComplaintType_COMPLAINT_TYPE_UNSPECIFIED,
					Feedback: &complaintv1.Feedback{},
					Body:     "test",
				},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Empty body",
			args: args{
				ctx: context.Background(),
				req: &complaintv1.RequestCreateComplaint{
					Type:     complaintv1.ComplaintType_COMPLAINT_TYPE_BUG,
					Feedback: &complaintv1.Feedback{},
					Body:     "",
				},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Nil feedback",
			args: args{
				ctx: context.Background(),
				req: &complaintv1.RequestCreateComplaint{
					Type:     complaintv1.ComplaintType_COMPLAINT_TYPE_BUG,
					Feedback: nil,
					Body:     "test",
				},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Usecase error",
			prepare: func(f *fields) {
				f.complaintUsecase.EXPECT().CreateUnAuthrozied(gomock.Any(), gomock.Any()).Return(nil, errors.New("internal error"))
			},
			args: args{
				ctx: context.Background(),
				req: &complaintv1.RequestCreateComplaint{
					Type: complaintv1.ComplaintType_COMPLAINT_TYPE_BUG,
					Feedback: &complaintv1.Feedback{
						FeedbackName:  "John Doe",
						FeedbackEmail: "john@example.com",
					},
					Body:   "Bug description",
					UserId: nil,
				},
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				complaintUsecase: mock.NewMockComplaintUsecase(ctrl),
				analyticUsecase:  mock.NewMockAnalyticUsecase(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ComplaintServer{
				complaintUsecase: f.complaintUsecase,
				analyticUsecase:  f.analyticUsecase,
				logger:           zap.NewNop(),
			}

			_, err := s.CreateComplaint(tt.args.ctx, tt.args.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tt.wantCode, st.Code())
		})
	}
}

func TestPositiveComplaintServer_GetComplaint(t *testing.T) {
	type fields struct {
		complaintUsecase *mock.MockComplaintUsecase
		analyticUsecase  *mock.MockAnalyticUsecase
	}

	type args struct {
		ctx context.Context
		req *complaintv1.RequestGetComplaint
	}

	now := time.Now()

	tests := []struct {
		prepare func(*fields)
		want    *complaintv1.ResponseGetComplaint
		name    string
		args    args
	}{
		{
			name: "Successful get complaint",
			prepare: func(f *fields) {
				f.complaintUsecase.EXPECT().GetComplaint(gomock.Any(), dtoComplaint.RequestGetComplaint{
					ComplaintId: 1,
				}).Return(dtoComplaint.ResponseGetComplaint{
					ComplaintDTO: dtoComplaint.ComplaintDTO{
						ID:     1,
						Type:   "bug",
						Status: "new",
						FeedbackDTO: dtoComplaint.FeedbackDTO{
							FeedBackName:  "John Doe",
							FeedBackEmail: "john@example.com",
						},
						Body:      "Bug description",
						UserID:    100,
						CreatedAt: now,
						UpdatedAt: now,
					},
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &complaintv1.RequestGetComplaint{ComplaintId: 1},
			},
			want: &complaintv1.ResponseGetComplaint{
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
					CreatedAt: timestamppb.New(now),
					UpdatedAt: timestamppb.New(now),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				complaintUsecase: mock.NewMockComplaintUsecase(ctrl),
				analyticUsecase:  mock.NewMockAnalyticUsecase(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ComplaintServer{
				complaintUsecase: f.complaintUsecase,
				analyticUsecase:  f.analyticUsecase,
				logger:           zap.NewNop(),
			}

			resp, err := s.GetComplaint(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want.GetComplaint().GetId(), resp.GetComplaint().GetId())
		})
	}
}

func TestNegativeComplaintServer_GetComplaint(t *testing.T) {
	type fields struct {
		complaintUsecase *mock.MockComplaintUsecase
		analyticUsecase  *mock.MockAnalyticUsecase
	}

	type args struct {
		ctx context.Context
		req *complaintv1.RequestGetComplaint
	}

	tests := []struct {
		wantCode codes.Code
		prepare  func(*fields)
		name     string
		args     args
	}{
		{
			name: "Nil request",
			args: args{
				ctx: context.Background(),
				req: nil,
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Invalid complaint_id",
			args: args{
				ctx: context.Background(),
				req: &complaintv1.RequestGetComplaint{ComplaintId: 0},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Complaint not found",
			prepare: func(f *fields) {
				f.complaintUsecase.EXPECT().GetComplaint(gomock.Any(), dtoComplaint.RequestGetComplaint{
					ComplaintId: 999,
				}).Return(dtoComplaint.ResponseGetComplaint{}, domainComplaint.ErrNotFound)
			},
			args: args{
				ctx: context.Background(),
				req: &complaintv1.RequestGetComplaint{ComplaintId: 999},
			},
			wantCode: codes.NotFound,
		},
		{
			name: "Usecase error",
			prepare: func(f *fields) {
				f.complaintUsecase.EXPECT().GetComplaint(gomock.Any(), dtoComplaint.RequestGetComplaint{
					ComplaintId: 1,
				}).Return(dtoComplaint.ResponseGetComplaint{}, errors.New("internal error"))
			},
			args: args{
				ctx: context.Background(),
				req: &complaintv1.RequestGetComplaint{ComplaintId: 1},
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				complaintUsecase: mock.NewMockComplaintUsecase(ctrl),
				analyticUsecase:  mock.NewMockAnalyticUsecase(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ComplaintServer{
				complaintUsecase: f.complaintUsecase,
				analyticUsecase:  f.analyticUsecase,
				logger:           zap.NewNop(),
			}

			_, err := s.GetComplaint(tt.args.ctx, tt.args.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tt.wantCode, st.Code())
		})
	}
}

func TestPositiveComplaintServer_GetComplaintsByUser(t *testing.T) {
	type fields struct {
		complaintUsecase *mock.MockComplaintUsecase
		analyticUsecase  *mock.MockAnalyticUsecase
	}

	type args struct {
		ctx context.Context
		req *complaintv1.RequestGetComplaintsByUser
	}

	now := time.Now()

	tests := []struct {
		prepare func(*fields)
		want    *complaintv1.ResponseGetComplaints
		name    string
		args    args
	}{
		{
			name: "Successful get complaints by user",
			prepare: func(f *fields) {
				f.complaintUsecase.EXPECT().GetComplaintsByUser(gomock.Any(), int64(100)).Return(dtoComplaint.ResponseGetComplaints{
					Complaints: []dtoComplaint.ComplaintDTO{
						{
							ID:     1,
							Type:   "bug",
							Status: "new",
							FeedbackDTO: dtoComplaint.FeedbackDTO{
								FeedBackName:  "John Doe",
								FeedBackEmail: "john@example.com",
							},
							Body:      "Bug description",
							UserID:    100,
							CreatedAt: now,
							UpdatedAt: now,
						},
					},
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &complaintv1.RequestGetComplaintsByUser{UserId: 100},
			},
			want: &complaintv1.ResponseGetComplaints{
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
						CreatedAt: timestamppb.New(now),
						UpdatedAt: timestamppb.New(now),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				complaintUsecase: mock.NewMockComplaintUsecase(ctrl),
				analyticUsecase:  mock.NewMockAnalyticUsecase(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ComplaintServer{
				complaintUsecase: f.complaintUsecase,
				analyticUsecase:  f.analyticUsecase,
				logger:           zap.NewNop(),
			}

			resp, err := s.GetComplaintsByUser(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Len(t, resp.GetComplaints(), 1)
			require.Equal(t, tt.want.GetComplaints()[0].GetId(), resp.GetComplaints()[0].GetId())
		})
	}
}

func TestNegativeComplaintServer_GetComplaintsByUser(t *testing.T) {
	type fields struct {
		complaintUsecase *mock.MockComplaintUsecase
		analyticUsecase  *mock.MockAnalyticUsecase
	}

	type args struct {
		ctx context.Context
		req *complaintv1.RequestGetComplaintsByUser
	}

	tests := []struct {
		wantCode codes.Code
		prepare  func(*fields)
		name     string
		args     args
	}{
		{
			name: "Nil request",
			args: args{
				ctx: context.Background(),
				req: nil,
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Invalid user_id",
			args: args{
				ctx: context.Background(),
				req: &complaintv1.RequestGetComplaintsByUser{UserId: 0},
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "Usecase error",
			prepare: func(f *fields) {
				f.complaintUsecase.EXPECT().GetComplaintsByUser(gomock.Any(), int64(100)).Return(dtoComplaint.ResponseGetComplaints{}, errors.New("internal error"))
			},
			args: args{
				ctx: context.Background(),
				req: &complaintv1.RequestGetComplaintsByUser{UserId: 100},
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				complaintUsecase: mock.NewMockComplaintUsecase(ctrl),
				analyticUsecase:  mock.NewMockAnalyticUsecase(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ComplaintServer{
				complaintUsecase: f.complaintUsecase,
				analyticUsecase:  f.analyticUsecase,
				logger:           zap.NewNop(),
			}

			_, err := s.GetComplaintsByUser(tt.args.ctx, tt.args.req)
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, tt.wantCode, st.Code())
		})
	}
}

func TestPositiveComplaintServer_GetAllComplaints(t *testing.T) {
	type fields struct {
		complaintUsecase *mock.MockComplaintUsecase
		analyticUsecase  *mock.MockAnalyticUsecase
	}

	type args struct {
		ctx context.Context
		req *complaintv1.RequestGetAllComplaints
	}

	now := time.Now()

	tests := []struct {
		prepare func(*fields)
		want    *complaintv1.ResponseGetComplaints
		name    string
		args    args
	}{
		{
			name: "Successful get all complaints",
			prepare: func(f *fields) {
				f.complaintUsecase.EXPECT().GetAllComplaints(gomock.Any()).Return(dtoComplaint.ResponseGetComplaints{
					Complaints: []dtoComplaint.ComplaintDTO{
						{
							ID:     1,
							Type:   "bug",
							Status: "new",
							FeedbackDTO: dtoComplaint.FeedbackDTO{
								FeedBackName:  "John Doe",
								FeedBackEmail: "john@example.com",
							},
							Body:      "Bug description",
							UserID:    100,
							CreatedAt: now,
							UpdatedAt: now,
						},
					},
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &complaintv1.RequestGetAllComplaints{},
			},
			want: &complaintv1.ResponseGetComplaints{
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
						CreatedAt: timestamppb.New(now),
						UpdatedAt: timestamppb.New(now),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				complaintUsecase: mock.NewMockComplaintUsecase(ctrl),
				analyticUsecase:  mock.NewMockAnalyticUsecase(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ComplaintServer{
				complaintUsecase: f.complaintUsecase,
				analyticUsecase:  f.analyticUsecase,
				logger:           zap.NewNop(),
			}

			resp, err := s.GetAllComplaints(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Len(t, resp.GetComplaints(), 1)
		})
	}
}

func TestPositiveComplaintServer_UpdateComplaintStatus(t *testing.T) {
	type fields struct {
		complaintUsecase *mock.MockComplaintUsecase
		analyticUsecase  *mock.MockAnalyticUsecase
	}

	type args struct {
		ctx context.Context
		req *complaintv1.RequestUpdateComplaintStatus
	}

	now := time.Now()

	tests := []struct {
		prepare func(*fields)
		want    *complaintv1.ResponseGetComplaint
		name    string
		args    args
	}{
		{
			name: "Successful update complaint status to in_progress",
			prepare: func(f *fields) {
				f.complaintUsecase.EXPECT().UpdateComplaintStatus(gomock.Any(), dtoComplaint.RequestUpdateComplaintStatus{
					ComplaintId: 1,
					Status:      "in_progress",
				}).Return(dtoComplaint.ResponseGetComplaint{
					ComplaintDTO: dtoComplaint.ComplaintDTO{
						ID:     1,
						Type:   "bug",
						Status: "in_progress",
						FeedbackDTO: dtoComplaint.FeedbackDTO{
							FeedBackName:  "John Doe",
							FeedBackEmail: "john@example.com",
						},
						Body:      "Bug description",
						UserID:    100,
						CreatedAt: now,
						UpdatedAt: now,
					},
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &complaintv1.RequestUpdateComplaintStatus{
					ComplaintId: 1,
					Status:      complaintv1.ComplaintStatus_COMPLAINT_STATUS_IN_PROGRESS,
				},
			},
			want: &complaintv1.ResponseGetComplaint{
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
					CreatedAt: timestamppb.New(now),
					UpdatedAt: timestamppb.New(now),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				complaintUsecase: mock.NewMockComplaintUsecase(ctrl),
				analyticUsecase:  mock.NewMockAnalyticUsecase(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ComplaintServer{
				complaintUsecase: f.complaintUsecase,
				analyticUsecase:  f.analyticUsecase,
				logger:           zap.NewNop(),
			}

			resp, err := s.UpdateComplaintStatus(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want.GetComplaint().GetId(), resp.GetComplaint().GetId())
			require.Equal(t, tt.want.GetComplaint().GetStatus(), resp.GetComplaint().GetStatus())
		})
	}
}

func TestPositiveComplaintServer_GetUserComplaintAnalytic(t *testing.T) {
	type fields struct {
		complaintUsecase *mock.MockComplaintUsecase
		analyticUsecase  *mock.MockAnalyticUsecase
	}

	type args struct {
		ctx context.Context
		req *complaintv1.RequestGetUserComplaintAnalytic
	}

	tests := []struct {
		prepare func(*fields)
		want    *complaintv1.ResponseGetUserComplaintAnalytic
		name    string
		args    args
	}{
		{
			name: "Successful get user complaint analytic",
			prepare: func(f *fields) {
				f.analyticUsecase.EXPECT().GetUserComplaintAnalytic(gomock.Any(), dtoAnalytic.RequestComplaintAnalytic{
					UserID: 100,
				}).Return(dtoAnalytic.ResponseComplaintAnalytic{
					CountStatus: dtoAnalytic.CountStatus{
						CountStatusOpened: 5,
						CountStatusInWork: 3,
						CountStatusClosed: 2,
					},
					CountType: dtoAnalytic.CountType{
						CountBug:     4,
						CountUpgrade: 3,
						CountProduct: 3,
					},
				}, nil)
			},
			args: args{
				ctx: context.Background(),
				req: &complaintv1.RequestGetUserComplaintAnalytic{UserId: 100},
			},
			want: &complaintv1.ResponseGetUserComplaintAnalytic{
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
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				complaintUsecase: mock.NewMockComplaintUsecase(ctrl),
				analyticUsecase:  mock.NewMockAnalyticUsecase(ctrl),
			}

			if tt.prepare != nil {
				tt.prepare(&f)
			}

			s := &ComplaintServer{
				complaintUsecase: f.complaintUsecase,
				analyticUsecase:  f.analyticUsecase,
				logger:           zap.NewNop(),
			}

			resp, err := s.GetUserComplaintAnalytic(tt.args.ctx, tt.args.req)
			require.NoError(t, err)
			require.Equal(t, tt.want.GetCountStatus().GetCountStatusOpened(), resp.GetCountStatus().GetCountStatusOpened())
			require.Equal(t, tt.want.GetCountType().GetCountBug(), resp.GetCountType().GetCountBug())
		})
	}
}

func ptrInt64(i int64) *int64 {
	return &i
}