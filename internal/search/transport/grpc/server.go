package grpc

import (
	"context"

	searchv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/search/v1"
	searchdto "github.com/go-park-mail-ru/2026_1_ASAP/internal/search/dto/search"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/loggerctx"
	"go.uber.org/zap"
)

type SearchUsecase interface {
	SearchChats(ctx context.Context, request *searchdto.SearchChatsRequest) (*searchdto.SearchChatsResponse, error)
	SearchGlobalChannels(ctx context.Context, request *searchdto.SearchGlobalChannelsRequest) (*searchdto.SearchGlobalChannelsResponse, error)
	SearchContacts(ctx context.Context, request *searchdto.SearchContactsRequest) (*searchdto.SearchContactsResponse, error)
	SearchMessagesInChat(ctx context.Context, request *searchdto.SearchMessagesInChatRequest) (*searchdto.SearchMessagesInChatResponse, error)
}

type SearchServer struct {
	searchv1.UnimplementedSearchServer

	searchUsecase SearchUsecase
	logger        *zap.Logger
}

func NewSearchServer(searchUsecase SearchUsecase, logger *zap.Logger) *SearchServer {
	return &SearchServer{
		searchUsecase: searchUsecase,
		logger:        logger,
	}
}

func (s *SearchServer) log(ctx context.Context) *zap.Logger {
	return loggerctx.EnrichLoggerFromContext(ctx, s.logger)
}

func (s *SearchServer) SearchChats(ctx context.Context, req *searchv1.SearchChatsRequest) (*searchv1.SearchChatsResponse, error) {
	_ = s.log(ctx)

	dtoReq := mapSearchChatsRequestProtoToDTO(req)
	resp, err := s.searchUsecase.SearchChats(ctx, dtoReq)
	if err != nil {
		return nil, mapDomainErrToProtoErr(err)
	}
	return mapSearchChatsResponseDTOToProto(resp), nil
}

func (s *SearchServer) SearchGlobalChannels(ctx context.Context, req *searchv1.SearchGlobalChannelsRequest) (*searchv1.SearchGlobalChannelsResponse, error) {
	_ = s.log(ctx)

	dtoReq := mapSearchGlobalChannelsRequestProtoToDTO(req)
	resp, err := s.searchUsecase.SearchGlobalChannels(ctx, dtoReq)
	if err != nil {
		return nil, mapDomainErrToProtoErr(err)
	}
	return mapSearchGlobalChannelsResponseDTOToProto(resp), nil
}

func (s *SearchServer) SearchContacts(ctx context.Context, req *searchv1.SearchContactsRequest) (*searchv1.SearchContactsResponse, error) {
	_ = s.log(ctx)

	dtoReq := mapSearchContactsRequestProtoToDTO(req)
	resp, err := s.searchUsecase.SearchContacts(ctx, dtoReq)
	if err != nil {
		return nil, mapDomainErrToProtoErr(err)
	}
	return mapSearchContactsResponseDTOToProto(resp), nil
}

func (s *SearchServer) SearchMessagesInChat(ctx context.Context, req *searchv1.SearchMessagesInChatRequest) (*searchv1.SearchMessagesInChatResponse, error) {
	_ = s.log(ctx)

	dtoReq := mapSearchMessagesInChatRequestProtoToDTO(req)
	resp, err := s.searchUsecase.SearchMessagesInChat(ctx, dtoReq)
	if err != nil {
		return nil, mapDomainErrToProtoErr(err)
	}
	return mapSearchMessagesInChatResponseDTOToProto(resp), nil
}
