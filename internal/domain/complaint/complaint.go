package complaint

import "time"

type ComplaintType string

const (
	ComplaintTypeBug     ComplaintType = "bug"
	ComplaintTypeProduct ComplaintType = "product"
	ComplaintTypeUpgrade ComplaintType = "upgrade"
)

type ComplaintStatus string

const (
	ComplaintStatusNew        ComplaintStatus = "new"
	ComplaintStatusInProgress ComplaintStatus = "in_progress"
	ComplaintStatusClosed     ComplaintStatus = "closed"
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
