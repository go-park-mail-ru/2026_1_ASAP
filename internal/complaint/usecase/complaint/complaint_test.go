package complaint

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/complaint/domain/complaint"
	dtoComplaint "github.com/go-park-mail-ru/2026_1_ASAP/internal/complaint/dto/complaint"
	dtoMedia "github.com/go-park-mail-ru/2026_1_ASAP/internal/complaint/dto/media"
	"github.com/stretchr/testify/require"
)

type complaintRepoStub struct {
	createFn               func(ctx context.Context, complaint domain.Complaint) (domain.Complaint, error)
	uploadAttachmentURLFn  func(ctx context.Context, complaintID int64, attachmentURL string) (domain.Complaint, error)
	getComplaintsByUserIDFn func(ctx context.Context, userID int64) ([]domain.Complaint, error)
	getComplaintByIDFn     func(ctx context.Context, id int64) (domain.Complaint, error)
	updateComplaintFn      func(ctx context.Context, complaintID int64, status domain.ComplaintStatus) (domain.Complaint, error)
	getAllComplaintsFn     func(ctx context.Context) ([]domain.Complaint, error)
}

func (s complaintRepoStub) Create(ctx context.Context, complaint domain.Complaint) (domain.Complaint, error) {
	return s.createFn(ctx, complaint)
}
func (s complaintRepoStub) UploadAttachmentURL(ctx context.Context, complaintID int64, attachmentURL string) (domain.Complaint, error) {
	return s.uploadAttachmentURLFn(ctx, complaintID, attachmentURL)
}
func (s complaintRepoStub) GetComplaintsByUserID(ctx context.Context, userID int64) ([]domain.Complaint, error) {
	return s.getComplaintsByUserIDFn(ctx, userID)
}
func (s complaintRepoStub) GetComplaintByID(ctx context.Context, id int64) (domain.Complaint, error) {
	return s.getComplaintByIDFn(ctx, id)
}
func (s complaintRepoStub) UpdateComplaint(ctx context.Context, complaintID int64, status domain.ComplaintStatus) (domain.Complaint, error) {
	return s.updateComplaintFn(ctx, complaintID, status)
}
func (s complaintRepoStub) GetAllComplaints(ctx context.Context) ([]domain.Complaint, error) {
	return s.getAllComplaintsFn(ctx)
}

type complaintMediaStub struct {
	uploadFn func(ctx context.Context, complaintID int64, input *dtoMedia.FileInput) (string, error)
}

func (s complaintMediaStub) UploadComplaint(ctx context.Context, complaintID int64, input *dtoMedia.FileInput) (string, error) {
	return s.uploadFn(ctx, complaintID, input)
}

func ptrString(v string) *string {
	return &v
}

func ptrInt64(v int64) *int64 {
	return &v
}

func sampleComplaint() domain.Complaint {
	now := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	return domain.Complaint{
		ID:            1,
		Type:          domain.ComplaintTypeBug,
		Status:        string(domain.ComplaintStatusNew),
		FeedBackName:  "Ann",
		FeedBackEmail: "ann@example.com",
		Body:          "Need fix",
		AttachmentURL: ptrString("https://file"),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func TestComplaintService_CreateUnAuthrozied_Positive(t *testing.T) {
	type fields struct {
		repo      ComplaintRepositoryInterface
		mediaRepo MediaRepositoryInterface
	}

	type args struct {
		ctx     context.Context
		request dtoComplaint.RequestCreateComplaint
	}

	file := &dtoMedia.FileInput{
		Body:        bytes.NewBufferString("file"),
		ContentType: "image/png",
		Size:        4,
	}

	tests := []struct {
		prepare func() fields
		want    *dtoComplaint.ResponseCreateComplaint
		name    string
		args    args
	}{
		{
			name: "create without file",
			prepare: func() fields {
				return fields{
					repo: complaintRepoStub{
						createFn: func(ctx context.Context, complaint domain.Complaint) (domain.Complaint, error) {
							require.Nil(t, complaint.UserID)
							require.Equal(t, "Ann", complaint.FeedBackName)
							out := sampleComplaint()
							out.AttachmentURL = nil
							return out, nil
						},
					},
				}
			},
			args: args{
				ctx: context.Background(),
				request: dtoComplaint.RequestCreateComplaint{
					Type: dtoComplaint.ComplaintType(domain.ComplaintTypeBug),
					FeedBackInfo: dtoComplaint.FeedbackDTO{
						FeedBackName:  "Ann",
						FeedBackEmail: "ann@example.com",
					},
					Body: "Need fix",
				},
			},
			want: &dtoComplaint.ResponseCreateComplaint{
				ComplaintDTO: dtoComplaint.ComplaintDTO{
					ID:     1,
					Type:   "bug",
					Status: "new",
					FeedbackDTO: dtoComplaint.FeedbackDTO{
						FeedBackName:  "Ann",
						FeedBackEmail: "ann@example.com",
					},
					Body:          "Need fix",
					UserID:        0,
					AttachmentURL: nil,
					CreatedAt:     sampleComplaint().CreatedAt,
					UpdatedAt:     sampleComplaint().UpdatedAt,
				},
			},
		},
		{
			name: "create with file",
			prepare: func() fields {
				return fields{
					repo: complaintRepoStub{
						createFn: func(ctx context.Context, complaint domain.Complaint) (domain.Complaint, error) {
							out := sampleComplaint()
							out.AttachmentURL = nil
							return out, nil
						},
						uploadAttachmentURLFn: func(ctx context.Context, complaintID int64, attachmentURL string) (domain.Complaint, error) {
							out := sampleComplaint()
							out.AttachmentURL = ptrString(attachmentURL)
							return out, nil
						},
					},
					mediaRepo: complaintMediaStub{
						uploadFn: func(ctx context.Context, complaintID int64, input *dtoMedia.FileInput) (string, error) {
							require.Equal(t, int64(1), complaintID)
							require.Equal(t, file, input)
							return "https://uploaded", nil
						},
					},
				}
			},
			args: args{
				ctx: context.Background(),
				request: dtoComplaint.RequestCreateComplaint{
					Type: dtoComplaint.ComplaintType(domain.ComplaintTypeBug),
					FeedBackInfo: dtoComplaint.FeedbackDTO{
						FeedBackName:  "Ann",
						FeedBackEmail: "ann@example.com",
					},
					Body: "Need fix",
					File: file,
				},
			},
			want: &dtoComplaint.ResponseCreateComplaint{
				ComplaintDTO: dtoComplaint.ComplaintDTO{
					ID:     1,
					Type:   "bug",
					Status: "new",
					FeedbackDTO: dtoComplaint.FeedbackDTO{
						FeedBackName:  "Ann",
						FeedBackEmail: "ann@example.com",
					},
					Body:          "Need fix",
					UserID:        0,
					AttachmentURL: ptrString("https://uploaded"),
					CreatedAt:     sampleComplaint().CreatedAt,
					UpdatedAt:     sampleComplaint().UpdatedAt,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.prepare()
			s := NewComplaintService(f.repo, f.mediaRepo)
			got, err := s.CreateUnAuthrozied(tt.args.ctx, tt.args.request)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestComplaintService_CreateUnAuthrozied_Negative(t *testing.T) {
	type fields struct {
		repo      ComplaintRepositoryInterface
		mediaRepo MediaRepositoryInterface
	}

	type args struct {
		ctx     context.Context
		request dtoComplaint.RequestCreateComplaint
	}

	file := &dtoMedia.FileInput{
		Body:        bytes.NewBufferString("file"),
		ContentType: "image/png",
		Size:        4,
	}

	tests := []struct {
		prepare    func() fields
		name       string
		args       args
		wantSubstr string
	}{
		{
			name: "create error",
			prepare: func() fields {
				return fields{
					repo: complaintRepoStub{
						createFn: func(ctx context.Context, complaint domain.Complaint) (domain.Complaint, error) {
							return domain.Complaint{}, errors.New("db down")
						},
					},
				}
			},
			args:       args{ctx: context.Background(), request: dtoComplaint.RequestCreateComplaint{Body: "x"}},
			wantSubstr: "create complaint",
		},
		{
			name: "media repo nil",
			prepare: func() fields {
				return fields{
					repo: complaintRepoStub{
						createFn: func(ctx context.Context, complaint domain.Complaint) (domain.Complaint, error) {
							out := sampleComplaint()
							out.AttachmentURL = nil
							return out, nil
						},
					},
				}
			},
			args: args{
				ctx: context.Background(),
				request: dtoComplaint.RequestCreateComplaint{
					Body: "x",
					File: file,
				},
			},
			wantSubstr: "media repository is nil",
		},
		{
			name: "upload error",
			prepare: func() fields {
				return fields{
					repo: complaintRepoStub{
						createFn: func(ctx context.Context, complaint domain.Complaint) (domain.Complaint, error) {
							out := sampleComplaint()
							out.AttachmentURL = nil
							return out, nil
						},
					},
					mediaRepo: complaintMediaStub{
						uploadFn: func(ctx context.Context, complaintID int64, input *dtoMedia.FileInput) (string, error) {
							return "", errors.New("s3 down")
						},
					},
				}
			},
			args: args{
				ctx: context.Background(),
				request: dtoComplaint.RequestCreateComplaint{
					Body: "x",
					File: file,
				},
			},
			wantSubstr: "upload complaint attachment",
		},
		{
			name: "save attachment url error",
			prepare: func() fields {
				return fields{
					repo: complaintRepoStub{
						createFn: func(ctx context.Context, complaint domain.Complaint) (domain.Complaint, error) {
							out := sampleComplaint()
							out.AttachmentURL = nil
							return out, nil
						},
						uploadAttachmentURLFn: func(ctx context.Context, complaintID int64, attachmentURL string) (domain.Complaint, error) {
							return domain.Complaint{}, errors.New("db update down")
						},
					},
					mediaRepo: complaintMediaStub{
						uploadFn: func(ctx context.Context, complaintID int64, input *dtoMedia.FileInput) (string, error) {
							return "https://uploaded", nil
						},
					},
				}
			},
			args: args{
				ctx: context.Background(),
				request: dtoComplaint.RequestCreateComplaint{
					Body: "x",
					File: file,
				},
			},
			wantSubstr: "save complaint attachment url",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.prepare()
			s := NewComplaintService(f.repo, f.mediaRepo)
			got, err := s.CreateUnAuthrozied(tt.args.ctx, tt.args.request)
			require.Nil(t, got)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantSubstr)
		})
	}
}

func TestComplaintService_CreateAuthrozied_Positive(t *testing.T) {
	type fields struct {
		repo      ComplaintRepositoryInterface
		mediaRepo MediaRepositoryInterface
	}

	type args struct {
		ctx     context.Context
		userID  int64
		request dtoComplaint.RequestCreateComplaint
	}

	tests := []struct {
		prepare func() fields
		want    *dtoComplaint.ResponseCreateComplaint
		name    string
		args    args
	}{
		{
			name: "create authorized complaint",
			prepare: func() fields {
				return fields{
					repo: complaintRepoStub{
						createFn: func(ctx context.Context, complaint domain.Complaint) (domain.Complaint, error) {
							require.Equal(t, ptrInt64(42), complaint.UserID)
							out := sampleComplaint()
							out.UserID = ptrInt64(42)
							out.AttachmentURL = nil
							return out, nil
						},
					},
				}
			},
			args: args{
				ctx:    context.Background(),
				userID: 42,
				request: dtoComplaint.RequestCreateComplaint{
					Type: dtoComplaint.ComplaintType(domain.ComplaintTypeBug),
					FeedBackInfo: dtoComplaint.FeedbackDTO{
						FeedBackName:  "Ann",
						FeedBackEmail: "ann@example.com",
					},
					Body: "Need fix",
				},
			},
			want: &dtoComplaint.ResponseCreateComplaint{
				ComplaintDTO: dtoComplaint.ComplaintDTO{
					ID:     1,
					Type:   "bug",
					Status: "new",
					FeedbackDTO: dtoComplaint.FeedbackDTO{
						FeedBackName:  "Ann",
						FeedBackEmail: "ann@example.com",
					},
					Body:          "Need fix",
					UserID:        42,
					AttachmentURL: nil,
					CreatedAt:     sampleComplaint().CreatedAt,
					UpdatedAt:     sampleComplaint().UpdatedAt,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.prepare()
			s := NewComplaintService(f.repo, f.mediaRepo)
			got, err := s.CreateAuthrozied(tt.args.ctx, tt.args.userID, tt.args.request)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestComplaintService_GetComplaintAndLists(t *testing.T) {
	type fields struct {
		repo      ComplaintRepositoryInterface
		mediaRepo MediaRepositoryInterface
	}

	type args struct {
		ctx         context.Context
		userID      int64
		complaintID int64
	}

	base := sampleComplaint()
	userComplaint := sampleComplaint()
	userComplaint.UserID = ptrInt64(99)

	tests := []struct {
		prepare func() fields
		name    string
		run     func(t *testing.T, s *ComplaintService, a args)
		args    args
	}{
		{
			name: "get complaint by id",
			prepare: func() fields {
				return fields{
					repo: complaintRepoStub{
						getComplaintByIDFn: func(ctx context.Context, id int64) (domain.Complaint, error) {
							return base, nil
						},
					},
				}
			},
			args: args{ctx: context.Background(), complaintID: 1},
			run: func(t *testing.T, s *ComplaintService, a args) {
				got, err := s.GetComplaint(a.ctx, dtoComplaint.RequestGetComplaint{ComplaintId: a.complaintID})
				require.NoError(t, err)
				require.Equal(t, int64(1), got.ComplaintDTO.ID)
				require.Equal(t, "bug", got.ComplaintDTO.Type)
			},
		},
		{
			name: "get complaints by user",
			prepare: func() fields {
				return fields{
					repo: complaintRepoStub{
						getComplaintsByUserIDFn: func(ctx context.Context, userID int64) ([]domain.Complaint, error) {
							return []domain.Complaint{userComplaint}, nil
						},
					},
				}
			},
			args: args{ctx: context.Background(), userID: 99},
			run: func(t *testing.T, s *ComplaintService, a args) {
				got, err := s.GetComplaintsByUser(a.ctx, a.userID)
				require.NoError(t, err)
				require.Len(t, got.Complaints, 1)
				require.Equal(t, int64(99), got.Complaints[0].UserID)
			},
		},
		{
			name: "get all complaints",
			prepare: func() fields {
				return fields{
					repo: complaintRepoStub{
						getAllComplaintsFn: func(ctx context.Context) ([]domain.Complaint, error) {
							return []domain.Complaint{base}, nil
						},
					},
				}
			},
			args: args{ctx: context.Background()},
			run: func(t *testing.T, s *ComplaintService, a args) {
				got, err := s.GetAllComplaints(a.ctx)
				require.NoError(t, err)
				require.Len(t, got.Complaints, 1)
				require.Equal(t, int64(1), got.Complaints[0].ID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.prepare()
			s := NewComplaintService(f.repo, f.mediaRepo)
			tt.run(t, s, tt.args)
		})
	}
}

func TestComplaintService_UpdateComplaintStatus(t *testing.T) {
	type fields struct {
		repo      ComplaintRepositoryInterface
		mediaRepo MediaRepositoryInterface
	}

	type args struct {
		ctx     context.Context
		request dtoComplaint.RequestUpdateComplaintStatus
	}

	tests := []struct {
		prepare    func() fields
		name       string
		args       args
		wantStatus string
		wantErr    error
		wantSubstr string
	}{
		{
			name: "success",
			prepare: func() fields {
				return fields{
					repo: complaintRepoStub{
						updateComplaintFn: func(ctx context.Context, complaintID int64, status domain.ComplaintStatus) (domain.Complaint, error) {
							out := sampleComplaint()
							out.Status = string(status)
							return out, nil
						},
					},
				}
			},
			args: args{
				ctx: context.Background(),
				request: dtoComplaint.RequestUpdateComplaintStatus{
					ComplaintId: 1,
					Status:      dtoComplaint.ComplaintStatus(domain.ComplaintStatusClosed),
				},
			},
			wantStatus: "closed",
		},
		{
			name: "not found",
			prepare: func() fields {
				return fields{
					repo: complaintRepoStub{
						updateComplaintFn: func(ctx context.Context, complaintID int64, status domain.ComplaintStatus) (domain.Complaint, error) {
							return domain.Complaint{}, domain.ErrNotFound
						},
					},
				}
			},
			args: args{
				ctx: context.Background(),
				request: dtoComplaint.RequestUpdateComplaintStatus{
					ComplaintId: 1,
					Status:      dtoComplaint.ComplaintStatus(domain.ComplaintStatusClosed),
				},
			},
			wantErr: domain.ErrNotFound,
		},
		{
			name: "repo error",
			prepare: func() fields {
				return fields{
					repo: complaintRepoStub{
						updateComplaintFn: func(ctx context.Context, complaintID int64, status domain.ComplaintStatus) (domain.Complaint, error) {
							return domain.Complaint{}, errors.New("db down")
						},
					},
				}
			},
			args: args{
				ctx: context.Background(),
				request: dtoComplaint.RequestUpdateComplaintStatus{
					ComplaintId: 1,
					Status:      dtoComplaint.ComplaintStatus(domain.ComplaintStatusClosed),
				},
			},
			wantSubstr: "update complaint status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.prepare()
			s := NewComplaintService(f.repo, f.mediaRepo)
			got, err := s.UpdateComplaintStatus(tt.args.ctx, tt.args.request)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			if tt.wantSubstr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantSubstr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantStatus, got.ComplaintDTO.Status)
		})
	}
}
