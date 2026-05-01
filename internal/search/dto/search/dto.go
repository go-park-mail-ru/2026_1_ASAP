package dto

import searchdomain "github.com/go-park-mail-ru/2026_1_ASAP/internal/search/domain/search"

type SearchChatsRequest struct {
	Query    string
	Kinds    []searchdomain.ChatType
	UserID   int64
	Limit    int32
	BeforeID int64
}

type SearchChatsResponse struct {
	Chats        []searchdomain.ChatHit
	NextBeforeID int64
}

type SearchContactsRequest struct {
	Query    string
	Scope    searchdomain.ContactScope
	UserID   int64
	Limit    int32
	BeforeID int64
}

type SearchContactsResponse struct {
	Contacts     []searchdomain.ContactHit
	NextBeforeID int64
}

type SearchMessagesInChatRequest struct {
	Query    string
	UserID   int64
	ChatID   int64
	Limit    int32
	BeforeID int64
}

type SearchMessagesInChatResponse struct {
	Messages     []searchdomain.MessageHit
	NextBeforeID int64
}

type SearchUsersRequest struct {
	Query        string
	CallerUserID int64
	Limit        int32
	BeforeID     int64
}

type SearchUsersResponse struct {
	Users        []searchdomain.ContactHit
	NextBeforeID int64
}
