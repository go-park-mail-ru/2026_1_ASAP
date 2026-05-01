package search

import (
	"context"
	"strings"
	"unicode/utf8"

	searchdomain "github.com/go-park-mail-ru/2026_1_ASAP/internal/search/domain/search"
	searchdto "github.com/go-park-mail-ru/2026_1_ASAP/internal/search/dto/search"
)

type SearchRepository interface {
	SearchChats(ctx context.Context, params *searchdomain.SearchChatsParams) (*searchdomain.SearchChatsResult, error)
	SearchContacts(ctx context.Context, params *searchdomain.SearchContactsParams) (*searchdomain.SearchContactsResult, error)
	SearchUsers(ctx context.Context, params *searchdomain.SearchUsersParams) (*searchdomain.SearchUsersResult, error)
	SearchMessagesInChat(ctx context.Context, params *searchdomain.SearchMessagesInChatParams) (*searchdomain.SearchMessagesInChatResult, error)
}

const (
	defaultLimit  = 20
	maxLimit      = 50
	maxQueryRunes = 256
)

type Service struct {
	repo SearchRepository
}

func NewService(repo SearchRepository) *Service {
	return &Service{repo: repo}
}

func clampLimit(limit int32) int32 {
	if limit <= 0 {
		return defaultLimit
	}
	if limit > maxLimit {
		return maxLimit
	}
	return limit
}

func normalizeQuery(q string) (string, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return "", searchdomain.ErrInvalidInput
	}
	if utf8.RuneCountInString(q) > maxQueryRunes {
		return "", searchdomain.ErrInvalidInput
	}
	return q, nil
}

func (s *Service) SearchChats(ctx context.Context, req *searchdto.SearchChatsRequest) (*searchdto.SearchChatsResponse, error) {
	if req == nil {
		return nil, searchdomain.ErrInvalidInput
	}
	if req.UserID <= 0 {
		return nil, searchdomain.ErrInvalidInput
	}

	q, err := normalizeQuery(req.Query)
	if err != nil {
		return nil, err
	}

	params := searchdomain.SearchChatsParams{
		UserID:   req.UserID,
		Query:    q,
		Kinds:    req.Kinds,
		Limit:    clampLimit(req.Limit),
		BeforeID: req.BeforeID,
	}
	res, err := s.repo.SearchChats(ctx, &params)
	if err != nil {
		return nil, err
	}
	return &searchdto.SearchChatsResponse{
		Chats:        res.Chats,
		NextBeforeID: res.NextBeforeID,
	}, nil
}

func (s *Service) SearchContacts(ctx context.Context, req *searchdto.SearchContactsRequest) (*searchdto.SearchContactsResponse, error) {
	if req == nil {
		return nil, searchdomain.ErrInvalidInput
	}
	if req.UserID <= 0 {
		return nil, searchdomain.ErrInvalidInput
	}
	if req.Scope != searchdomain.ContactScopeContacts && req.Scope != searchdomain.ContactScopeLocal {
		return nil, searchdomain.ErrInvalidInput
	}

	q, err := normalizeQuery(req.Query)
	if err != nil {
		return nil, err
	}

	params := searchdomain.SearchContactsParams{
		UserID:   req.UserID,
		Query:    q,
		Scope:    req.Scope,
		Limit:    clampLimit(req.Limit),
		BeforeID: req.BeforeID,
	}
	res, err := s.repo.SearchContacts(ctx, &params)
	if err != nil {
		return nil, err
	}
	return &searchdto.SearchContactsResponse{
		Contacts:     res.Contacts,
		NextBeforeID: res.NextBeforeID,
	}, nil
}

func (s *Service) SearchUsers(ctx context.Context, req *searchdto.SearchUsersRequest) (*searchdto.SearchUsersResponse, error) {
	if req == nil {
		return nil, searchdomain.ErrInvalidInput
	}
	if req.CallerUserID <= 0 {
		return nil, searchdomain.ErrInvalidInput
	}

	q, err := normalizeQuery(req.Query)
	if err != nil {
		return nil, err
	}

	params := searchdomain.SearchUsersParams{
		RequesterID: req.CallerUserID,
		Query:       q,
		Limit:       clampLimit(req.Limit),
		BeforeID:    req.BeforeID,
	}
	res, err := s.repo.SearchUsers(ctx, &params)
	if err != nil {
		return nil, err
	}
	return &searchdto.SearchUsersResponse{
		Users:        res.Users,
		NextBeforeID: res.NextBeforeID,
	}, nil
}

func (s *Service) SearchMessagesInChat(ctx context.Context, req *searchdto.SearchMessagesInChatRequest) (*searchdto.SearchMessagesInChatResponse, error) {
	if req == nil {
		return nil, searchdomain.ErrInvalidInput
	}
	if req.UserID <= 0 || req.ChatID <= 0 {
		return nil, searchdomain.ErrInvalidInput
	}

	q, err := normalizeQuery(req.Query)
	if err != nil {
		return nil, err
	}

	params := searchdomain.SearchMessagesInChatParams{
		UserID:   req.UserID,
		ChatID:   req.ChatID,
		Query:    q,
		Limit:    clampLimit(req.Limit),
		BeforeID: req.BeforeID,
	}
	res, err := s.repo.SearchMessagesInChat(ctx, &params)
	if err != nil {
		return nil, err
	}
	return &searchdto.SearchMessagesInChatResponse{
		Messages:     res.Messages,
		NextBeforeID: res.NextBeforeID,
	}, nil
}
