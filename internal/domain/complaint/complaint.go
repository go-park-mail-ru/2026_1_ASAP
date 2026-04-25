package complaint

import "time"

type ComplaintType string

const (
	ComplaintTypeBug     ComplaintType = "bug"
	ComplaintTypeProduct ComplaintType = "product"
	ComplaintTypeUpgrade ComplaintType = "upgrade"
)

type Complaint struct {
	ID            int64
	Status        string
	Type          ComplaintType
	FeedBackName  string
	FeedBackEmail string
	Body          string
	UserID        *int64
	AttachmentURL *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
