package chat

import (
	"time"
)

type ChatType string

const (
	ChatTypeDialog  ChatType = "dialog"
	ChatTypeGroup   ChatType = "group"
	ChatTypeChannel ChatType = "channel"
)

type Chat struct {
	CreatedAt         time.Time
	UpdatedAt         time.Time
	Description       *string
	AvatarUrl         *string
	Type              ChatType
	Title             string
	Id                int64
	OwnerId           int64
	UnreadCount       int64
	LastReadMessageID int64
}

type AttachmentType string

const (
	AttachmentTypePhoto   AttachmentType = "photo"
	AttachmentTypeVideo   AttachmentType = "video"
	AttachmentTypeFile    AttachmentType = "file"
	AttachmentTypeContact AttachmentType = "contact"
	AttachmentTypeVoice   AttachmentType = "voice"
)

type Message struct {
	CreatedAt   time.Time
	UpdatedAt   time.Time
	StickerId   *int64
	DeletedAt   *time.Time
	Content     string
	Id          int64
	ChatId      int64
	SenderId    int64
	Edited      bool
	Attachments []MessageAttachment
}

type MessageAttachment struct {
	CreatedAt        time.Time
	FileURL          *string
	FileName         *string
	MimeType         *string
	ContactAvatarURL *string
	ContactFirstName *string
	ContactLastName  *string
	Waveform         []uint8
	Type             AttachmentType
	Id               int64
	MessageId        int64
	ContactUserID    *int64
	FileSize         *int64
	DurationMs       *int32
	SortOrder        int
}
