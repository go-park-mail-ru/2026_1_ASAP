package search

import "time"

type ChatType string

const (
	ChatTypeDialog  ChatType = "dialog"
	ChatTypeGroup   ChatType = "group"
	ChatTypeChannel ChatType = "channel"
)

type ContactScope string

const (
	ContactScopeContacts ContactScope = "contacts"
	ContactScopeLocal    ContactScope = "local"
)

type ChatHit struct {
	LastMessageAt      *time.Time
	AvatarURL          *string
	LastMessagePreview *string
	Title              string
	Type               ChatType
	ChatID             int64
	UnreadCount        int64
}

type ContactHit struct {
	LastSeenAt  *time.Time
	Login       *string
	AvatarURL   *string
	DisplayName string
	UserID      int64
	IsOnline    bool
}

type MessageHighlight struct {
	Fragment string
}

type MessageHit struct {
	CreatedAt   time.Time
	Highlights  []MessageHighlight
	TextPreview string
	MessageID   int64
	ChatID      int64
	SenderID    int64
}

type SearchChatsParams struct {
	Query    string
	Kinds    []ChatType
	UserID   int64
	Limit    int32
	BeforeID int64
}

type SearchChatsResult struct {
	Chats        []ChatHit
	NextBeforeID int64
}

type GlobalChannelHit struct {
	LastMessageAt      *time.Time
	AvatarURL          *string
	LastMessagePreview *string
	Title              string
	ChatID             int64
	IsMember           bool
}

type SearchGlobalChannelsParams struct {
	UserID   int64
	Query    string
	Limit    int32
	BeforeID int64
}

type SearchGlobalChannelsResult struct {
	Channels     []GlobalChannelHit
	NextBeforeID int64
}

type SearchContactsParams struct {
	Query    string
	Scope    ContactScope
	UserID   int64
	Limit    int32
	BeforeID int64
}

type SearchContactsResult struct {
	Contacts     []ContactHit
	NextBeforeID int64
}

type SearchMessagesInChatParams struct {
	Query    string
	UserID   int64
	ChatID   int64
	Limit    int32
	BeforeID int64
}

type SearchMessagesInChatResult struct {
	Messages     []MessageHit
	NextBeforeID int64
}

type SearchUsersParams struct {
	RequesterID int64
	Query       string
	Limit       int32
	BeforeID    int64
}

type SearchUsersResult struct {
	Users        []ContactHit
	NextBeforeID int64
}
