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
