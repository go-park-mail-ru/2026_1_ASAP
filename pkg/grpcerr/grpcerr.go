package grpcerr

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	commonv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/common/v1"
)

func New(grpcCode codes.Code, appCode int32, msg string) error {
	st, err := status.New(grpcCode, msg).WithDetails(&commonv1.Error{
		Code:    appCode,
		Message: msg,
	})
	if err != nil {
		return status.Error(grpcCode, msg)
	}
	return st.Err()
}

func Error(err error) (codes.Code, int32, string) {
	st, ok := status.FromError(err)
	if !ok {
		return codes.Internal, 0, err.Error()
	}
	for _, detail := range st.Details() {
		if e, ok := detail.(*commonv1.Error); ok {
			return st.Code(), e.Code, e.Message
		}
	}
	return st.Code(), 0, st.Message()
}
