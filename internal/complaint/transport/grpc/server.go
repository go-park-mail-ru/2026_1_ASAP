package grpc

import (
	"bytes"
	"context"
	"errors"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"

	complaintv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/complaint/v1"
	domainComplaint "github.com/go-park-mail-ru/2026_1_ASAP/internal/complaint/domain/complaint"
	dtoAnalytic "github.com/go-park-mail-ru/2026_1_ASAP/internal/complaint/dto/analytic"
	dtoComplaint "github.com/go-park-mail-ru/2026_1_ASAP/internal/complaint/dto/complaint"
	dtoMedia "github.com/go-park-mail-ru/2026_1_ASAP/internal/complaint/dto/media"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/grpcerr"
	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/loggerctx"
)

type ComplaintUsecase interface {
	CreateUnAuthrozied(ctx context.Context, request dtoComplaint.RequestCreateComplaint) (*dtoComplaint.ResponseCreateComplaint, error)
	CreateAuthrozied(ctx context.Context, userID int64, request dtoComplaint.RequestCreateComplaint) (*dtoComplaint.ResponseCreateComplaint, error)
	GetComplaint(ctx context.Context, request dtoComplaint.RequestGetComplaint) (dtoComplaint.ResponseGetComplaint, error)
	GetComplaintsByUser(ctx context.Context, userID int64) (dtoComplaint.ResponseGetComplaints, error)
	GetAllComplaints(ctx context.Context) (dtoComplaint.ResponseGetComplaints, error)
	UpdateComplaintStatus(ctx context.Context, request dtoComplaint.RequestUpdateComplaintStatus) (dtoComplaint.ResponseGetComplaint, error)
}

type AnalyticUsecase interface {
	GetUserComplaintAnalytic(ctx context.Context, request dtoAnalytic.RequestComplaintAnalytic) (dtoAnalytic.ResponseComplaintAnalytic, error)
}

type ComplaintServer struct {
	complaintv1.UnimplementedComplaintServer

	complaintUsecase ComplaintUsecase
	analyticUsecase  AnalyticUsecase
	logger           *zap.Logger
}

func NewComplaintServer(complaintUsecase ComplaintUsecase, analyticUsecase AnalyticUsecase, logger *zap.Logger) *ComplaintServer {
	return &ComplaintServer{
		complaintUsecase: complaintUsecase,
		analyticUsecase:  analyticUsecase,
		logger:           logger,
	}
}

func (s *ComplaintServer) CreateComplaint(ctx context.Context, req *complaintv1.RequestCreateComplaint) (*complaintv1.ResponseCreateComplaint, error) {
	if req == nil {
		return nil, grpcerr.New(codes.InvalidArgument, int32(complaintv1.ComplaintErrorCode_COMPLAINT_ERROR_INVALID_INPUT), "request is required")
	}
	if req.GetType() == complaintv1.ComplaintType_COMPLAINT_TYPE_UNSPECIFIED || req.GetBody() == "" || req.GetFeedback() == nil {
		return nil, grpcerr.New(codes.InvalidArgument, int32(complaintv1.ComplaintErrorCode_COMPLAINT_ERROR_INVALID_INPUT), "type, feedback and body are required")
	}

	requestDTO := dtoComplaint.RequestCreateComplaint{
		Type: dtoComplaint.ComplaintType(mapComplaintTypeToString(req.GetType())),
		FeedBackInfo: dtoComplaint.FeedbackDTO{
			FeedBackName:  req.GetFeedback().GetFeedbackName(),
			FeedBackEmail: req.GetFeedback().GetFeedbackEmail(),
		},
		Body: req.GetBody(),
	}

	if req.File != nil && len(req.GetFile().GetContent()) > 0 {
		requestDTO.File = &dtoMedia.FileInput{
			Body:        bytes.NewReader(req.GetFile().GetContent()),
			ContentType: req.GetFile().GetType(),
			Size:        int64(len(req.GetFile().GetContent())),
		}
	}

	var (
		resp *dtoComplaint.ResponseCreateComplaint
		err  error
	)
	if req.UserId != nil && req.GetUserId() > 0 {
		resp, err = s.complaintUsecase.CreateAuthrozied(ctx, req.GetUserId(), requestDTO)
	} else {
		resp, err = s.complaintUsecase.CreateUnAuthrozied(ctx, requestDTO)
	}
	if err != nil {
		s.log(ctx).Error("create complaint failed", zap.Error(err))
		return nil, grpcerr.New(codes.Internal, int32(complaintv1.ComplaintErrorCode_COMPLAINT_ERROR_INTERNAL), "failed to create complaint")
	}

	return &complaintv1.ResponseCreateComplaint{
		Complaint: mapComplaintToProto(resp.ComplaintDTO),
	}, nil
}

func (s *ComplaintServer) GetComplaint(ctx context.Context, req *complaintv1.RequestGetComplaint) (*complaintv1.ResponseGetComplaint, error) {
	if req == nil || req.GetComplaintId() <= 0 {
		return nil, grpcerr.New(codes.InvalidArgument, int32(complaintv1.ComplaintErrorCode_COMPLAINT_ERROR_INVALID_INPUT), "complaint_id is required")
	}

	resp, err := s.complaintUsecase.GetComplaint(ctx, dtoComplaint.RequestGetComplaint{ComplaintId: req.GetComplaintId()})
	if err != nil {
		if errors.Is(err, domainComplaint.ErrNotFound) {
			return nil, grpcerr.New(codes.NotFound, int32(complaintv1.ComplaintErrorCode_COMPLAINT_ERROR_NOT_FOUND), "complaint not found")
		}
		s.log(ctx).Error("get complaint failed", zap.Int64("complaint_id", req.GetComplaintId()), zap.Error(err))
		return nil, grpcerr.New(codes.Internal, int32(complaintv1.ComplaintErrorCode_COMPLAINT_ERROR_INTERNAL), "failed to get complaint")
	}

	return &complaintv1.ResponseGetComplaint{Complaint: mapComplaintToProto(resp.ComplaintDTO)}, nil
}

func (s *ComplaintServer) GetComplaintsByUser(ctx context.Context, req *complaintv1.RequestGetComplaintsByUser) (*complaintv1.ResponseGetComplaints, error) {
	if req == nil || req.GetUserId() <= 0 {
		return nil, grpcerr.New(codes.InvalidArgument, int32(complaintv1.ComplaintErrorCode_COMPLAINT_ERROR_INVALID_INPUT), "user_id is required")
	}

	resp, err := s.complaintUsecase.GetComplaintsByUser(ctx, req.GetUserId())
	if err != nil {
		s.log(ctx).Error("get complaints by user failed", zap.Int64("user_id", req.GetUserId()), zap.Error(err))
		return nil, grpcerr.New(codes.Internal, int32(complaintv1.ComplaintErrorCode_COMPLAINT_ERROR_INTERNAL), "failed to get complaints by user")
	}

	return &complaintv1.ResponseGetComplaints{Complaints: mapComplaintsToProto(resp.Complaints)}, nil
}

func (s *ComplaintServer) GetAllComplaints(ctx context.Context, _ *complaintv1.RequestGetAllComplaints) (*complaintv1.ResponseGetComplaints, error) {
	resp, err := s.complaintUsecase.GetAllComplaints(ctx)
	if err != nil {
		s.log(ctx).Error("get all complaints failed", zap.Error(err))
		return nil, grpcerr.New(codes.Internal, int32(complaintv1.ComplaintErrorCode_COMPLAINT_ERROR_INTERNAL), "failed to get complaints")
	}

	return &complaintv1.ResponseGetComplaints{Complaints: mapComplaintsToProto(resp.Complaints)}, nil
}

func (s *ComplaintServer) UpdateComplaintStatus(ctx context.Context, req *complaintv1.RequestUpdateComplaintStatus) (*complaintv1.ResponseGetComplaint, error) {
	if req == nil || req.GetComplaintId() <= 0 || req.GetStatus() == complaintv1.ComplaintStatus_COMPLAINT_STATUS_UNSPECIFIED {
		return nil, grpcerr.New(codes.InvalidArgument, int32(complaintv1.ComplaintErrorCode_COMPLAINT_ERROR_INVALID_INPUT), "complaint_id and status are required")
	}

	resp, err := s.complaintUsecase.UpdateComplaintStatus(ctx, dtoComplaint.RequestUpdateComplaintStatus{
		ComplaintId: req.GetComplaintId(),
		Status:      dtoComplaint.ComplaintStatus(mapComplaintStatusToString(req.GetStatus())),
	})
	if err != nil {
		if errors.Is(err, domainComplaint.ErrNotFound) {
			return nil, grpcerr.New(codes.NotFound, int32(complaintv1.ComplaintErrorCode_COMPLAINT_ERROR_NOT_FOUND), "complaint not found")
		}
		s.log(ctx).Error("update complaint status failed", zap.Int64("complaint_id", req.GetComplaintId()), zap.Error(err))
		return nil, grpcerr.New(codes.Internal, int32(complaintv1.ComplaintErrorCode_COMPLAINT_ERROR_INTERNAL), "failed to update complaint status")
	}

	return &complaintv1.ResponseGetComplaint{Complaint: mapComplaintToProto(resp.ComplaintDTO)}, nil
}

func (s *ComplaintServer) GetUserComplaintAnalytic(ctx context.Context, req *complaintv1.RequestGetUserComplaintAnalytic) (*complaintv1.ResponseGetUserComplaintAnalytic, error) {
	if req == nil || req.GetUserId() <= 0 {
		return nil, grpcerr.New(codes.InvalidArgument, int32(complaintv1.ComplaintErrorCode_COMPLAINT_ERROR_INVALID_INPUT), "user_id is required")
	}

	resp, err := s.analyticUsecase.GetUserComplaintAnalytic(ctx, dtoAnalytic.RequestComplaintAnalytic{UserID: req.GetUserId()})
	if err != nil {
		s.log(ctx).Error("get user complaint analytic failed", zap.Int64("user_id", req.GetUserId()), zap.Error(err))
		return nil, grpcerr.New(codes.Internal, int32(complaintv1.ComplaintErrorCode_COMPLAINT_ERROR_INTERNAL), "failed to get complaint analytic")
	}

	return &complaintv1.ResponseGetUserComplaintAnalytic{
		CountStatus: &complaintv1.CountStatus{
			CountStatusOpened: resp.CountStatus.CountStatusOpened,
			CountStatusInWork: resp.CountStatus.CountStatusInWork,
			CountStatusClosed: resp.CountStatus.CountStatusClosed,
		},
		CountType: &complaintv1.CountType{
			CountBug:     resp.CountType.CountBug,
			CountUpgrade: resp.CountType.CountUpgrade,
			CountProduct: resp.CountType.CountProduct,
		},
	}, nil
}

func (s *ComplaintServer) log(ctx context.Context) *zap.Logger {
	return loggerctx.EnrichLoggerFromContext(ctx, s.logger)
}
