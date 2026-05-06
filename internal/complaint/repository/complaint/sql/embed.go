package complaintsql

import _ "embed"

//go:embed insert_complaint.sql
var InsertComplaint string

//go:embed get_complaint_by_id.sql
var GetComplaintByID string

//go:embed get_complaints_by_user_id.sql
var GetComplaintsByUserID string

//go:embed upload_attachment_url.sql
var UploadAttachmentURL string

//go:embed update_complaint_status.sql
var UpdateComplaintStatus string

//go:embed get_all.sql
var GetAll string
