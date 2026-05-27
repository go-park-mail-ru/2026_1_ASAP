package complaint

type createComplaintRequest struct {
	Type     string   `json:"type"`
	Feedback feedback `json:"feedback"`
	Body     string   `json:"body"`
}

type updateStatusRequest struct {
	ComplaintID int64  `json:"complaint_id"`
	Status      string `json:"status"`
}

type complaintDTO struct {
	ID            int64    `json:"id"`
	Type          string   `json:"type"`
	Status        string   `json:"status"`
	Feedback      feedback `json:"feedback"`
	Body          string   `json:"body"`
	UserID        int64    `json:"user_id,omitempty"`
	AttachmentURL *string  `json:"attachment_url,omitempty"`
	CreatedAt     anyTime  `json:"created_at"`
	UpdatedAt     anyTime  `json:"updated_at"`
}

type feedback struct {
	FeedbackName  string `json:"feedback_name"`
	FeedbackEmail string `json:"feedback_email"`
}

type complaintAnalyticDTO struct {
	CountStatus countStatusDTO `json:"count_status"`
	CountType   countTypeDTO   `json:"count_type"`
}

type countStatusDTO struct {
	CountStatusOpened int64 `json:"count_status_opened"`
	CountStatusInWork int64 `json:"count_status_in_work"`
	CountStatusClosed int64 `json:"count_status_closed"`
}

type countTypeDTO struct {
	CountBug     int64 `json:"count_type_bug"`
	CountUpgrade int64 `json:"count_type_upgrade"`
	CountProduct int64 `json:"count_type_product"`
}

type anyTime = string
