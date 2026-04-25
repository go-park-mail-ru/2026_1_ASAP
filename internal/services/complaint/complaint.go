package complaint

import (
	"context"
	"fmt"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/complaint"
	dtoComplaint "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/complaint"
	"github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/media"
)

type MediaRepositoryInterface interface {
	UploadComplaint(ctx context.Context, complaintID int64, input *media.FileInput) (string, error)
}

type ComplaintRepositoryInterface interface {
	Create(ctx context.Context, complaint domain.Complaint) (domain.Complaint, error)
	UploadAttachmentURL(ctx context.Context, complaintID int64, attachmentURL string) (domain.Complaint, error)
	GetComplaintsByUserID(ctx context.Context, userID int64) ([]domain.Complaint, error)
	GetComplaintByID(ctx context.Context, id int64) (domain.Complaint, error)
	UpdateComplaint(ctx context.Context, complaintID int64, status domain.ComplaintType) (domain.Complaint, error)
}

type ComplaintService struct {
	repo      ComplaintRepositoryInterface
	mediaRepo MediaRepositoryInterface
}

func (c ComplaintService) CreateUnAuthrozied(ctx context.Context, request dtoComplaint.RequestCreateComplaint) (*dtoComplaint.ResponseCreateComplaint, error) {
	complaint, err := c.repo.Create(ctx, domain.Complaint{
		Type:          domain.ComplaintType(request.Type),
		FeedBackName:  request.FeedBackInfo.FeedBackName,
		FeedBackEmail: request.FeedBackInfo.FeedBackEmail,
		Body:          request.Body,
	})
	if err != nil {
		return nil, fmt.Errorf("create complaint: %w", err)
	}

	if request.File != nil {
		if c.mediaRepo == nil {
			return nil, fmt.Errorf("upload complaint attachment: media repository is nil")
		}
		attachmentURL, err := c.mediaRepo.UploadComplaint(ctx, complaint.ID, request.File)
		if err != nil {
			return nil, fmt.Errorf("upload complaint attachment: %w", err)
		}
		complaint, err = c.repo.UploadAttachmentURL(ctx, complaint.ID, attachmentURL)
		if err != nil {
			return nil, fmt.Errorf("save complaint attachment url: %w", err)
		}
	}

	return &dtoComplaint.ResponseCreateComplaint{
		ComplaintDTO: complaintToDTO(complaint),
	}, nil
}

func (c ComplaintService) CreateAuthrozied(ctx context.Context, userID int64, request dtoComplaint.RequestCreateComplaint) (*dtoComplaint.ResponseCreateComplaint, error) {
	complaint, err := c.repo.Create(ctx, domain.Complaint{
		Type:          domain.ComplaintType(request.Type),
		FeedBackName:  request.FeedBackInfo.FeedBackName,
		FeedBackEmail: request.FeedBackInfo.FeedBackEmail,
		Body:          request.Body,
		UserID:        &userID,
	})
	if err != nil {
		return nil, fmt.Errorf("create complaint: %w", err)
	}

	if request.File != nil {
		if c.mediaRepo == nil {
			return nil, fmt.Errorf("upload complaint attachment: media repository is nil")
		}
		attachmentURL, err := c.mediaRepo.UploadComplaint(ctx, complaint.ID, request.File)
		if err != nil {
			return nil, fmt.Errorf("upload complaint attachment: %w", err)
		}
		complaint, err = c.repo.UploadAttachmentURL(ctx, complaint.ID, attachmentURL)
		if err != nil {
			return nil, fmt.Errorf("save complaint attachment url: %w", err)
		}
	}

	return &dtoComplaint.ResponseCreateComplaint{
		ComplaintDTO: complaintToDTO(complaint),
	}, nil
}

func (c ComplaintService) GetComplaint(ctx context.Context, request dtoComplaint.RequestGetComplaint) (dtoComplaint.ResponseGetComplaint, error) {
	complaint, err := c.repo.GetComplaintByID(ctx, request.ComplaintId)
	if err != nil {
		return dtoComplaint.ResponseGetComplaint{}, fmt.Errorf("get complaint by id: %w", err)
	}

	return dtoComplaint.ResponseGetComplaint{
		ComplaintDTO: complaintToDTO(complaint),
	}, nil
}

func (c ComplaintService) GetComplaintsByUser(ctx context.Context, userID int64) (dtoComplaint.ResponseGetComplaints, error) {
	complaints, err := c.repo.GetComplaintsByUserID(ctx, userID)
	if err != nil {
		return dtoComplaint.ResponseGetComplaints{}, fmt.Errorf("get complaints by user id: %w", err)
	}

	response := dtoComplaint.ResponseGetComplaints{
		Complaints: make([]dtoComplaint.ComplaintDTO, 0, len(complaints)),
	}
	for _, complaint := range complaints {
		response.Complaints = append(response.Complaints, complaintToDTO(complaint))
	}

	return response, nil
}

func (c ComplaintService) UpdateComplaintStatus(ctx context.Context, request dtoComplaint.RequestUpdateComplaintStatus) (dtoComplaint.ResponseGetComplaint, error) {
	complaint, err := c.repo.UpdateComplaint(ctx, request.ComplaintId, domain.ComplaintType(request.Status))
	if err != nil {
		return dtoComplaint.ResponseGetComplaint{}, fmt.Errorf("update complaint status: %w", err)
	}

	return dtoComplaint.ResponseGetComplaint{
		ComplaintDTO: complaintToDTO(complaint),
	}, nil
}

func NewComplaintService(repo ComplaintRepositoryInterface, mediaRepo MediaRepositoryInterface) *ComplaintService {
	return &ComplaintService{
		repo:      repo,
		mediaRepo: mediaRepo,
	}
}

func complaintToDTO(complaint domain.Complaint) dtoComplaint.ComplaintDTO {
	var userID int64
	if complaint.UserID != nil {
		userID = *complaint.UserID
	}

	return dtoComplaint.ComplaintDTO{
		ID:     complaint.ID,
		Type:   string(complaint.Type),
		Status: complaint.Status,
		FeedbackDTO: dtoComplaint.FeedbackDTO{
			FeedBackName:  complaint.FeedBackName,
			FeedBackEmail: complaint.FeedBackEmail,
		},
		Body:          complaint.Body,
		UserID:        userID,
		AttachmentURL: complaint.AttachmentURL,
		CreatedAt:     complaint.CreatedAt,
		UpdatedAt:     complaint.UpdatedAt,
	}
}
