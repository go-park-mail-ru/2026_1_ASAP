package message

type AttachmentInput struct {
	Type          string `json:"type"`
	URL           string `json:"url,omitempty"`
	FileName      string `json:"file_name,omitempty"`
	ContactUserID int64  `json:"contact_user_id,omitempty"`
}

type RequestSendMessageAttachments struct {
	ChatID      int64             `json:"chat_id"`
	Text        string            `json:"text,omitempty"`
	Attachments []AttachmentInput `json:"attachments"`
}

type MessageAttachmentDTO struct {
	Type             string  `json:"type"`
	URL              *string `json:"url,omitempty"`
	FileName         *string `json:"file_name,omitempty"`
	MimeType         *string `json:"mime_type,omitempty"`
	FileSize         *int64  `json:"file_size,omitempty"`
	DurationMs       *int32  `json:"duration_ms,omitempty"`
	Waveform         []uint8 `json:"waveform,omitempty"`
	Transcript       *string `json:"transcript,omitempty"`
	CanTranscribe    *bool   `json:"can_transcribe,omitempty"`
	ContactUserID    *int64  `json:"contact_user_id,omitempty"`
	ContactFirstName *string `json:"contact_first_name,omitempty"`
	ContactLastName  *string `json:"contact_last_name,omitempty"`
	ContactAvatarURL *string `json:"contact_avatar_url,omitempty"`
}

type RequestTranscribeVoice struct {
	ChatID    int64 `json:"chat_id"`
	MessageID int64 `json:"message_id"`
}

type ResponseVoiceTranscript struct {
	ChatID       int64  `json:"chat_id"`
	MessageID    int64  `json:"message_id"`
	AttachmentID int64  `json:"attachment_id"`
	Transcript   string `json:"transcript"`
}

type UploadAttachmentResponse struct {
	AttachmentURL string  `json:"attachment_url"`
	ObjectKey     string  `json:"object_key,omitempty"`
	MimeType      string  `json:"mime_type"`
	FileSize      int64   `json:"file_size"`
	FileName      *string `json:"file_name,omitempty"`
	DurationMs    int32   `json:"duration_ms,omitempty"`
	Waveform      []uint8 `json:"waveform,omitempty"`
}
