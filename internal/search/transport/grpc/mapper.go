package grpc

import (
	"errors"

	searchv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/search/v1"
	searchdomain "github.com/go-park-mail-ru/2026_1_ASAP/internal/search/domain/search"
	searchdto "github.com/go-park-mail-ru/2026_1_ASAP/internal/search/dto/search"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/grpcerr"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func mapDomainErrToProtoErr(err error) error {
	switch {
	case errors.Is(err, searchdomain.ErrInvalidInput):
		return grpcerr.New(codes.InvalidArgument, int32(searchv1.SearchErrorCode_SEARCH_ERROR_INVALID_INPUT), err.Error())
	case errors.Is(err, searchdomain.ErrNotFound):
		return grpcerr.New(codes.NotFound, int32(searchv1.SearchErrorCode_SEARCH_ERROR_NOT_FOUND), err.Error())
	case errors.Is(err, searchdomain.ErrForbidden):
		return grpcerr.New(codes.PermissionDenied, int32(searchv1.SearchErrorCode_SEARCH_ERROR_PERMISSION_DENIED), err.Error())
	case errors.Is(err, searchdomain.ErrInternal):
		return grpcerr.New(codes.Internal, int32(searchv1.SearchErrorCode_SEARCH_ERROR_INTERNAL), err.Error())
	default:
		if err == nil {
			return nil
		}
		return grpcerr.New(codes.Internal, int32(searchv1.SearchErrorCode_SEARCH_ERROR_INTERNAL), err.Error())
	}
}

func mapSearchChatKindsProtoToDomain(kinds []searchv1.SearchChatKind) []searchdomain.ChatType {
	if len(kinds) == 0 {
		return nil
	}

	out := make([]searchdomain.ChatType, 0, len(kinds))
	for _, k := range kinds {
		switch k {
		case searchv1.SearchChatKind_SEARCH_CHAT_KIND_DIALOG:
			out = append(out, searchdomain.ChatTypeDialog)
		case searchv1.SearchChatKind_SEARCH_CHAT_KIND_GROUP:
			out = append(out, searchdomain.ChatTypeGroup)
		case searchv1.SearchChatKind_SEARCH_CHAT_KIND_CHANNEL:
			out = append(out, searchdomain.ChatTypeChannel)
		case searchv1.SearchChatKind_SEARCH_CHAT_KIND_UNSPECIFIED:
		default:
			// ignore unknown
		}
	}
	return out
}

func mapContactScopeProtoToDomain(scope searchv1.SearchContactScope) searchdomain.ContactScope {
	switch scope {
	case searchv1.SearchContactScope_SEARCH_CONTACT_SCOPE_CONTACTS:
		return searchdomain.ContactScopeContacts
	case searchv1.SearchContactScope_SEARCH_CONTACT_SCOPE_LOCAL:
		return searchdomain.ContactScopeLocal
	default:
		return ""
	}
}

func mapChatTypeDomainToProto(t searchdomain.ChatType) searchv1.ChatType {
	switch t {
	case searchdomain.ChatTypeDialog:
		return searchv1.ChatType_CHAT_TYPE_DIALOG
	case searchdomain.ChatTypeGroup:
		return searchv1.ChatType_CHAT_TYPE_GROUP
	case searchdomain.ChatTypeChannel:
		return searchv1.ChatType_CHAT_TYPE_CHANNEL
	default:
		return searchv1.ChatType_CHAT_TYPE_UNSPECIFIED
	}
}

func mapSearchGlobalChannelsRequestProtoToDTO(req *searchv1.SearchGlobalChannelsRequest) *searchdto.SearchGlobalChannelsRequest {
	if req == nil {
		return nil
	}
	return &searchdto.SearchGlobalChannelsRequest{
		UserID:   req.GetUserId(),
		Query:    req.GetQuery(),
		Limit:    req.GetLimit(),
		BeforeID: req.GetBeforeId(),
	}
}

func mapSearchGlobalChannelsResponseDTOToProto(resp *searchdto.SearchGlobalChannelsResponse) *searchv1.SearchGlobalChannelsResponse {
	if resp == nil {
		return &searchv1.SearchGlobalChannelsResponse{}
	}
	out := &searchv1.SearchGlobalChannelsResponse{
		NextBeforeId: resp.NextBeforeID,
		Channels:     make([]*searchv1.SearchGlobalChannelItem, 0, len(resp.Channels)),
	}
	for i := range resp.Channels {
		out.Channels = append(out.Channels, mapGlobalChannelHitDomainToProto(&resp.Channels[i]))
	}
	return out
}

func mapGlobalChannelHitDomainToProto(hit *searchdomain.GlobalChannelHit) *searchv1.SearchGlobalChannelItem {
	if hit == nil {
		return nil
	}
	item := &searchv1.SearchGlobalChannelItem{
		ChatId:   hit.ChatID,
		Title:    hit.Title,
		IsMember: hit.IsMember,
	}
	if hit.AvatarURL != nil {
		item.AvatarUrl = proto.String(*hit.AvatarURL)
	}
	if hit.LastMessagePreview != nil {
		item.LastMessagePreview = proto.String(*hit.LastMessagePreview)
	}
	if hit.LastMessageAt != nil {
		item.LastMessageAt = timestamppb.New(*hit.LastMessageAt)
	}
	return item
}

func mapSearchChatsRequestProtoToDTO(req *searchv1.SearchChatsRequest) *searchdto.SearchChatsRequest {
	if req == nil {
		return nil
	}
	return &searchdto.SearchChatsRequest{
		UserID:   req.GetUserId(),
		Query:    req.GetQuery(),
		Kinds:    mapSearchChatKindsProtoToDomain(req.GetKinds()),
		Limit:    req.GetLimit(),
		BeforeID: req.GetBeforeId(),
	}
}

func mapSearchChatsResponseDTOToProto(resp *searchdto.SearchChatsResponse) *searchv1.SearchChatsResponse {
	if resp == nil {
		return &searchv1.SearchChatsResponse{}
	}
	out := &searchv1.SearchChatsResponse{
		NextBeforeId: resp.NextBeforeID,
		Chats:        make([]*searchv1.SearchChatItem, 0, len(resp.Chats)),
	}
	for i := range resp.Chats {
		out.Chats = append(out.Chats, mapChatHitDomainToProto(&resp.Chats[i]))
	}
	return out
}

func mapChatHitDomainToProto(hit *searchdomain.ChatHit) *searchv1.SearchChatItem {
	if hit == nil {
		return nil
	}
	item := &searchv1.SearchChatItem{
		ChatId:      hit.ChatID,
		Type:        mapChatTypeDomainToProto(hit.Type),
		Title:       hit.Title,
		UnreadCount: hit.UnreadCount,
	}
	if hit.AvatarURL != nil {
		item.AvatarUrl = proto.String(*hit.AvatarURL)
	}
	if hit.LastMessagePreview != nil {
		item.LastMessagePreview = proto.String(*hit.LastMessagePreview)
	}
	if hit.LastMessageAt != nil {
		item.LastMessageAt = timestamppb.New(*hit.LastMessageAt)
	}
	return item
}

func mapSearchContactsRequestProtoToDTO(req *searchv1.SearchContactsRequest) *searchdto.SearchContactsRequest {
	if req == nil {
		return nil
	}
	return &searchdto.SearchContactsRequest{
		UserID:   req.GetUserId(),
		Query:    req.GetQuery(),
		Scope:    mapContactScopeProtoToDomain(req.GetScope()),
		Limit:    req.GetLimit(),
		BeforeID: req.GetBeforeId(),
	}
}

func mapSearchContactsResponseDTOToProto(resp *searchdto.SearchContactsResponse) *searchv1.SearchContactsResponse {
	if resp == nil {
		return &searchv1.SearchContactsResponse{}
	}
	out := &searchv1.SearchContactsResponse{
		NextBeforeId: resp.NextBeforeID,
		Contacts:     make([]*searchv1.SearchContactItem, 0, len(resp.Contacts)),
	}
	for i := range resp.Contacts {
		out.Contacts = append(out.Contacts, mapContactHitDomainToProto(&resp.Contacts[i]))
	}
	return out
}

func mapContactHitDomainToProto(hit *searchdomain.ContactHit) *searchv1.SearchContactItem {
	if hit == nil {
		return nil
	}
	item := &searchv1.SearchContactItem{
		UserId:      hit.UserID,
		DisplayName: hit.DisplayName,
		IsOnline:    hit.IsOnline,
	}
	if hit.Login != nil {
		item.Login = proto.String(*hit.Login)
	}
	if hit.AvatarURL != nil {
		item.AvatarUrl = proto.String(*hit.AvatarURL)
	}
	if hit.LastSeenAt != nil {
		item.LastSeenAt = timestamppb.New(*hit.LastSeenAt)
	}
	return item
}

func mapSearchMessagesInChatRequestProtoToDTO(req *searchv1.SearchMessagesInChatRequest) *searchdto.SearchMessagesInChatRequest {
	if req == nil {
		return nil
	}
	return &searchdto.SearchMessagesInChatRequest{
		UserID:   req.GetUserId(),
		ChatID:   req.GetChatId(),
		Query:    req.GetQuery(),
		Limit:    req.GetLimit(),
		BeforeID: req.GetBeforeId(),
	}
}

func mapSearchMessagesInChatResponseDTOToProto(resp *searchdto.SearchMessagesInChatResponse) *searchv1.SearchMessagesInChatResponse {
	if resp == nil {
		return &searchv1.SearchMessagesInChatResponse{}
	}
	out := &searchv1.SearchMessagesInChatResponse{
		NextBeforeId: resp.NextBeforeID,
		Messages:     make([]*searchv1.SearchMessageItem, 0, len(resp.Messages)),
	}
	for i := range resp.Messages {
		out.Messages = append(out.Messages, mapMessageHitDomainToProto(&resp.Messages[i]))
	}
	return out
}

func mapMessageHighlightsDomainToProto(highlights []searchdomain.MessageHighlight) *searchv1.SearchMessageHighlight {
	for _, h := range highlights {
		if h.Fragment != "" {
			return &searchv1.SearchMessageHighlight{Fragment: h.Fragment}
		}
	}
	return nil
}

func mapMessageHitDomainToProto(hit *searchdomain.MessageHit) *searchv1.SearchMessageItem {
	if hit == nil {
		return nil
	}
	item := &searchv1.SearchMessageItem{
		MessageId:   hit.MessageID,
		ChatId:      hit.ChatID,
		SenderId:    hit.SenderID,
		TextPreview: hit.TextPreview,
		CreatedAt:   timestamppb.New(hit.CreatedAt),
	}
	item.Highlights = mapMessageHighlightsDomainToProto(hit.Highlights)
	return item
}
