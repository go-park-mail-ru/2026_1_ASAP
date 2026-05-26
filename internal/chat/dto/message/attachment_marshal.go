package message

import "encoding/json"

type messageAttachmentWire struct {
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
	IsBlur           *bool   `json:"is_blur,omitempty"`
}

func (a MessageAttachmentDTO) MarshalJSON() ([]byte, error) {
	wire := messageAttachmentWire{
		Type:             a.Type,
		URL:              a.URL,
		FileName:         a.FileName,
		MimeType:         a.MimeType,
		FileSize:         a.FileSize,
		DurationMs:       a.DurationMs,
		Waveform:         a.Waveform,
		Transcript:       a.Transcript,
		CanTranscribe:    a.CanTranscribe,
		ContactUserID:    a.ContactUserID,
		ContactFirstName: a.ContactFirstName,
		ContactLastName:  a.ContactLastName,
		ContactAvatarURL: a.ContactAvatarURL,
	}
	if a.Type == "photo" {
		blur := a.IsBlur
		wire.IsBlur = &blur
	}
	return json.Marshal(wire)
}
