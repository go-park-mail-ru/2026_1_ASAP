package complaint

import "github.com/go-park-mail-ru/2026_1_ASAP/internal/complaint/dto/media"

type ComplaintType string
type ComplaintStatus string

const (
	ComplaintTypeBug     ComplaintType = "bug"
	ComplaintTypeProduct ComplaintType = "product"
	ComplaintTypeUpgrade ComplaintType = "upgrade"
)

const (
	ComplaintStatusNew        ComplaintStatus = "new"
	ComplaintStatusInProgress ComplaintStatus = "in_progress"
	ComplaintStatusClosed     ComplaintStatus = "closed"
)

type RequestCreateComplaint struct {
	Type         ComplaintType `json:"type"`
	FeedBackInfo FeedbackDTO   `json:"feedback"`
	Body         string        `json:"body"`
	File         *media.FileInput
}
