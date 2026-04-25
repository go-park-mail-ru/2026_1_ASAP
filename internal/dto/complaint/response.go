package complaint

import "time"

type ComplaintDTO struct {
	Type          string      `json:"type"`
	Status        string      `json:"status"`
	FeedbackDTO   FeedbackDTO `json:"feedback"`
	Body          string      `json:"body"`
	UserID        int64       `json:"user_id"`
	AttachmentURL *string     `json:"attachment_url,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

type ResponseCreateComplaint struct {
	ComplaintDTO ComplaintDTO `json:"complaint"`
}

type ResponseGetComplaint struct {
	ComplaintDTO ComplaintDTO `json:"complaint"`
}

type ResponseGetComplaints struct {
	Complaints []ComplaintDTO `json:"complaints"`
}
