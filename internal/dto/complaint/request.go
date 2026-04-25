package complaint

import "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/media"

type ComplaintType string

const (
	ComplaintTypeBug     ComplaintType = "bug"
	ComplaintTypeProduct ComplaintType = "product"
	ComplaintTypeUpgrade ComplaintType = "upgrade"
)

type RequestCreateComplaint struct {
	Type         ComplaintType `json:"type"`
	FeedBackInfo FeedbackDTO   `json:"feedback"`
	Body         string        `json:"body"`
	File         *media.FileInput
}

type RequestGetComplaint struct {
	ComplaintId int64 `json:"complaint_id"`
}

type RequestGetComplaints struct {
	UserID int64 `json:"user_id"`
}

type FeedbackDTO struct {
	FeedBackName  string `json:"feedback_name"`
	FeedBackEmail string `json:"feedback_email"`
}
