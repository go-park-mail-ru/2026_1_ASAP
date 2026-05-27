package complaint

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

type RequestUpdateComplaintStatus struct {
	ComplaintId int64           `json:"complaint_id"`
	Status      ComplaintStatus `json:"status"`
}
